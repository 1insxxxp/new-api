package controller

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSyncPublicGroupsOnceUsesSourceAndTargetRoutes(t *testing.T) {
	const secret = "test-secret"
	payload := []byte(`{"version":1,"snapshots":[]}`)
	var fetched, applied bool
	verifySignature := func(r *http.Request, body []byte) bool {
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(r.Header.Get("X-Public-Group-Sync-Timestamp") + "\n"))
		_, _ = mac.Write(body)
		return r.Header.Get("X-Public-Group-Sync-Signature") == hex.EncodeToString(mac.Sum(nil))
	}
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/internal/public-group-sync/snapshot" {
			http.NotFound(w, r)
			return
		}
		if !verifySignature(r, nil) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fetched = true
		_, _ = w.Write(payload)
	}))
	defer source.Close()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/internal/sub-public-group-sync/snapshot" {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(body, payload) || !verifySignature(r, body) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		applied = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	if err := syncPublicGroupsOnce(source.Client(), source.URL, target.URL, secret); err != nil {
		t.Fatal(err)
	}
	if !fetched || !applied {
		t.Fatalf("incomplete sync: fetched=%v applied=%v", fetched, applied)
	}
}

func TestSignedPublicGroupSyncRequestUsesTimestampAndBody(t *testing.T) {
	const secret = "test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("X-Public-Group-Sync-Timestamp")
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(timestamp))
		_, _ = mac.Write([]byte("\n"))
		_, _ = mac.Write(body)
		if got := r.Header.Get("X-Public-Group-Sync-Signature"); got != hex.EncodeToString(mac.Sum(nil)) {
			t.Fatalf("signature mismatch: got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if _, err := signedPublicGroupSyncRequest(context.Background(), server.Client(), http.MethodPost, server.URL, []byte(`{"ok":true}`), secret); err != nil {
		t.Fatal(err)
	}
}
