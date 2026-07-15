package hint

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveBaseURL(t *testing.T) {
	prevTesting := Testing
	defer func() { Testing = prevTesting }()

	t.Run("DefaultsToProduction", func(t *testing.T) {
		Testing = false
		if got := getBackend("").resolveBaseURL(); got != ProdAPIURL {
			t.Fatalf("expected %q, got %q", ProdAPIURL, got)
		}
	})

	t.Run("DefaultsToStagingWhenTesting", func(t *testing.T) {
		Testing = true
		if got := getBackend("").resolveBaseURL(); got != StagingAPIURL {
			t.Fatalf("expected %q, got %q", StagingAPIURL, got)
		}
	})

	t.Run("PrefersConfiguredBaseURL", func(t *testing.T) {
		// Configured base URL should win regardless of the Testing flag.
		const custom = "https://api.example.com/api"
		Testing = true
		if got := getBackend(custom).resolveBaseURL(); got != custom {
			t.Fatalf("expected %q, got %q", custom, got)
		}
	})
}

func TestNewRequestUsesConfiguredBaseURL(t *testing.T) {
	const custom = "https://api.example.com/api"
	req, err := getBackend(custom).NewRequest("GET", "/provider/practitioners", "key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := custom + "/provider/practitioners"; req.URL.String() != want {
		t.Fatalf("expected request URL %q, got %q", want, req.URL.String())
	}
}

func TestWithBaseURLRoutesCalls(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	// Point the client at the test server via the option. Set Testing to a
	// value whose default URL would differ, to prove the option takes effect.
	prevTesting := Testing
	Testing = true
	defer func() { Testing = prevTesting }()

	client := NewPracticeClient("access-token", WithBaseURL(srv.URL))
	if _, err := client.ListAllPractitioners(); err != nil {
		t.Fatal(err)
	}

	if want := "/provider/practitioners"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
}
