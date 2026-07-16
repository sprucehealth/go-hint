package hint

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newGrantServer(t *testing.T, gotPath, gotKey *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotPath = r.URL.Path
		*gotKey, _, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"grant-id","access_token":"practice-key"}`))
	}))
}

func TestNewOAuthClientWithBaseURL(t *testing.T) {
	prevKey := Key
	defer func() { Key = prevKey }()
	Key = "global-key"

	var gotPath, gotKey string
	srv := newGrantServer(t, &gotPath, &gotKey)
	defer srv.Close()

	client := NewOAuthClient(WithBaseURL(srv.URL))
	grant, err := client.GrantAPIKey("auth-code")
	if err != nil {
		t.Fatal(err)
	}

	if want := "/oauth/tokens"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
	// Without WithPartnerKey the exchange authenticates with the global Key.
	if want := "global-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
	if want := "practice-key"; grant.AccessToken != want {
		t.Fatalf("expected access token %q, got %q", want, grant.AccessToken)
	}
}

func TestNewOAuthClientWithPartnerKey(t *testing.T) {
	prevKey := Key
	defer func() { Key = prevKey }()
	Key = "global-key"

	var gotPath, gotKey string
	srv := newGrantServer(t, &gotPath, &gotKey)
	defer srv.Close()

	client := NewOAuthClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	if _, err := client.GrantAPIKey("auth-code"); err != nil {
		t.Fatal(err)
	}

	// The per-client key should win over the global Key.
	if want := "sandbox-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
}

func TestNewOAuthClientDefaults(t *testing.T) {
	prevTesting := Testing
	prevKey := Key
	defer func() {
		Testing = prevTesting
		Key = prevKey
	}()
	Testing = true
	Key = "global-key"

	client, ok := NewOAuthClient().(*oauthClient)
	if !ok {
		t.Fatalf("expected *oauthClient, got %T", NewOAuthClient())
	}

	// With no options the base URL falls back to the Testing-flag default and
	// the key falls back to the global Key at call time.
	backend, ok := client.B.(BackendConfiguration)
	if !ok {
		t.Fatalf("expected BackendConfiguration, got %T", client.B)
	}
	if got := backend.resolveBaseURL(); got != StagingAPIURL {
		t.Fatalf("expected %q, got %q", StagingAPIURL, got)
	}
	if want := "global-key"; client.resolveKey() != want {
		t.Fatalf("expected key %q, got %q", want, client.resolveKey())
	}
}
