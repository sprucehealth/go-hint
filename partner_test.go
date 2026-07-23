package hint

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newPartnerServer records the basic-auth username (the partner key) used on the
// request and returns a minimal partner body.
func newPartnerServer(t *testing.T, gotKey *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotKey, _, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Acme"}`))
	}))
}

func TestPartnerClientHonorsExplicitKey(t *testing.T) {
	// Guards the fix: the key passed to NewPartnerClient used to be dropped
	// because Get/Update read the global Key directly.
	prevKey := Key
	defer func() { Key = prevKey }()
	Key = "global-key"

	var gotKey string
	srv := newPartnerServer(t, &gotKey)
	defer srv.Close()

	client := NewPartnerClient(getBackend(srv.URL), "explicit-key")
	if _, err := client.Get(); err != nil {
		t.Fatal(err)
	}
	if want := "explicit-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
}

func TestPartnerClientFallsBackToGlobalKey(t *testing.T) {
	prevKey := Key
	defer func() { Key = prevKey }()
	Key = "global-key"

	var gotKey string
	srv := newPartnerServer(t, &gotKey)
	defer srv.Close()

	// No per-client key: authentication falls back to the global Key, preserving
	// the default-client behavior of GetPartner/UpdatePartner.
	client := NewPartnerClient(getBackend(srv.URL), "")
	if _, err := client.Get(); err != nil {
		t.Fatal(err)
	}
	if want := "global-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
}
