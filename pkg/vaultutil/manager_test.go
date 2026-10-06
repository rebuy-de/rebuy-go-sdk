package vaultutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func TestManagerCloseWaitsForRevoke(t *testing.T) {
	// The revocation must finish before Close returns, even though the
	// context passed to Init is already cancelled. Otherwise the process
	// exits while the request is still in flight.
	var revocations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/token/revoke-self" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		select {
		case <-time.After(200 * time.Millisecond):
		case <-r.Context().Done():
			t.Error("revoke request got cancelled")
			return
		}

		revocations.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	conf := api.DefaultConfig()
	conf.Address = server.URL
	client, err := api.NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken("test-token")

	watcher, err := client.NewLifetimeWatcher(&api.LifetimeWatcherInput{
		Secret: &api.Secret{Auth: &api.SecretAuth{ClientToken: "test-token", Renewable: true}},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := start(ctx, client, Params{}, watcher)

	err = m.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if revocations.Load() != 1 {
		t.Fatal("Close returned before the token got revoked")
	}

	// A second Close must not block or revoke again.
	err = m.Close()
	if err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if n := revocations.Load(); n != 1 {
		t.Fatalf("expected exactly one revocation, got %d", n)
	}
}
