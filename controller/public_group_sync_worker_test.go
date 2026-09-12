package controller

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
