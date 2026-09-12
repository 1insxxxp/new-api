package controller

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const publicGroupSyncPath = "/api/internal/sub-public-group-sync/snapshot"

// StartPublicGroupSyncWorker starts the optional Sub -> New bridge. It is a
// no-op unless both the source URL and shared secret are configured.
func StartPublicGroupSyncWorker() {
	source := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_GROUP_SYNC_SOURCE_URL")), "/")
	secret := strings.TrimSpace(os.Getenv("PUBLIC_GROUP_SYNC_SECRET"))
	if source == "" || secret == "" {
		return
	}
	interval := 5 * time.Second
	if raw := strings.TrimSpace(os.Getenv("PUBLIC_GROUP_SYNC_INTERVAL_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 1 {
			interval = time.Duration(seconds) * time.Second
		}
	}
	target := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_GROUP_SYNC_TARGET_URL")), "/")
	if target == "" {
		port := strings.TrimSpace(os.Getenv("PORT"))
		if port == "" {
			port = "3000"
		}
		target = "http://127.0.0.1:" + port
	}
	go runPublicGroupSyncWorker(source, target, secret, interval)
}

func runPublicGroupSyncWorker(source, target, secret string, interval time.Duration) {
	client := &http.Client{Timeout: 15 * time.Second}
	for {
		if err := syncPublicGroupsOnce(client, source, target, secret); err != nil {
			log.Printf("public group sync failed: %v", err)
		}
		time.Sleep(interval)
	}
}

func syncPublicGroupsOnce(client *http.Client, source, target, secret string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	payload, err := signedPublicGroupSyncRequest(ctx, client, http.MethodGet, source+publicGroupSyncPath, nil, secret)
	if err != nil {
		return fmt.Errorf("fetch snapshot: %w", err)
	}
	_, err = signedPublicGroupSyncRequest(ctx, client, http.MethodPost, target+publicGroupSyncPath, payload, secret)
	if err != nil {
		return fmt.Errorf("apply snapshot: %w", err)
	}
	return nil
}

func signedPublicGroupSyncRequest(ctx context.Context, client *http.Client, method, endpoint string, body []byte, secret string) ([]byte, error) {
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Public-Group-Sync-Timestamp", timestamp)
	req.Header.Set("X-Public-Group-Sync-Signature", hex.EncodeToString(mac.Sum(nil)))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}
