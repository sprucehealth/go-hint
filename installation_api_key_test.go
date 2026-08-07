package hint

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

// apiKeyJSON is the single-key shape, as returned by create and update.
const apiKeyJSON = `{
	"id": "sbx-ktesttoken000000000000000000000000",
	"token": "sbx-ktesttoken000000000000000000000000",
	"label": "Production",
	"created_at": "2026-07-16T20:15:25.529087Z",
	"last_used_at": "2026-07-20T15:39:20.000000Z",
	"deactivated_at": null
}`

// collectAPIKeys drains an API key iterator into a slice, failing the test if
// the iterator reports an error.
func collectAPIKeys(t *testing.T, iter *APIKeyIter) []*APIKey {
	t.Helper()

	var apiKeys []*APIKey
	for iter.Next() {
		apiKeys = append(apiKeys, iter.APIKey())
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}
	return apiKeys
}

func TestAPIKeyListParamsEncode(t *testing.T) {
	cases := []struct {
		name     string
		params   *APIKeyListParams
		expected string
	}{
		{"nil", nil, ""},
		{"empty", &APIKeyListParams{}, ""},
		{"label", &APIKeyListParams{Label: "Production"}, "label=Production"},
		{
			"deactivated at is present",
			&APIKeyListParams{DeactivatedAt: []*Operation{IsPresent(true)}},
			"deactivated_at=%7B%22is_present%22%3Atrue}",
		},
		{
			"deactivated at is not present",
			&APIKeyListParams{DeactivatedAt: []*Operation{IsPresent(false)}},
			"deactivated_at=%7B%22is_present%22%3Afalse}",
		},
		{
			"all params",
			&APIKeyListParams{
				Label:         "Production",
				CreatedAt:     []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
				LastUsedAt:    []*Operation{{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"}},
				DeactivatedAt: []*Operation{IsPresent(false)},
				Offset:        10,
				Limit:         25,
			},
			"label=Production" +
				"&created_at=%7B%22gte%22%3A%222026-01-01T00%3A00%3A00Z%22}" +
				"&last_used_at=%7B%22lt%22%3A%222026-02-01T00%3A00%3A00Z%22}" +
				"&deactivated_at=%7B%22is_present%22%3Afalse}" +
				"&offset=10&limit=25",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.params.Encode()
			if err != nil {
				t.Fatal(err)
			}
			if encoded != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, encoded)
			}
		})
	}
}

func TestInstallationClientListAPIKeys(t *testing.T) {
	var gotPath, gotQuery, gotMethod, gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotMethod = r.Method
		gotKey, _, _ = r.BasicAuth()
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		// The list endpoint returns key metadata only: no id, no token.
		w.Write([]byte(`[
			{
				"label": "Production",
				"token_last_4": "0000",
				"created_at": "2026-07-16T20:15:25.529087Z",
				"last_used_at": "2026-07-20T15:39:20.000000Z",
				"deactivated_at": null
			},
			{
				"label": null,
				"token_last_4": "abcd",
				"created_at": "2026-07-16T20:15:25.529087Z",
				"last_used_at": null,
				"deactivated_at": "2026-07-25T12:00:00.000000Z"
			}
		]`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	apiKeys := collectAPIKeys(t, client.ListAPIKeys("sbx-inst-aaaa1111bbbb", nil))

	if want := "/partner/installations/sbx-inst-aaaa1111bbbb/api_keys"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("expected no query string for nil params, got %q", gotQuery)
	}
	if want := "GET"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := "sandbox-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}

	if len(apiKeys) != 2 {
		t.Fatalf("expected 2 api keys, got %d", len(apiKeys))
	}

	live := apiKeys[0]
	if live.Label == nil || *live.Label != "Production" {
		t.Fatalf("unexpected label: %v", live.Label)
	}
	if live.TokenLast4 != "0000" {
		t.Fatalf("unexpected token_last_4: %q", live.TokenLast4)
	}
	if live.ID != "" || live.Token != "" {
		t.Fatalf("expected the list endpoint to return no id/token, got %q / %q", live.ID, live.Token)
	}
	if live.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be parsed")
	}
	if live.LastUsedAt == nil || live.LastUsedAt.IsZero() {
		t.Fatalf("expected last_used_at to be parsed, got %v", live.LastUsedAt)
	}
	if live.DeactivatedAt != nil {
		t.Fatalf("expected deactivated_at to be nil, got %v", live.DeactivatedAt)
	}

	deactivated := apiKeys[1]
	if deactivated.Label != nil {
		t.Fatalf("expected nil label, got %v", deactivated.Label)
	}
	if deactivated.LastUsedAt != nil {
		t.Fatalf("expected nil last_used_at, got %v", deactivated.LastUsedAt)
	}
	if deactivated.DeactivatedAt == nil || deactivated.DeactivatedAt.IsZero() {
		t.Fatalf("expected deactivated_at to be parsed, got %v", deactivated.DeactivatedAt)
	}
}

func TestListAPIKeysSendsFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	iter := client.ListAPIKeys("inst_1", &APIKeyListParams{
		Label:         "Production",
		CreatedAt:     []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
		LastUsedAt:    []*Operation{{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"}},
		DeactivatedAt: []*Operation{IsPresent(true)},
		Limit:         25,
	})
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}

	// The decoded query is asserted so the test documents what Hint receives
	// rather than the escaping.
	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ key, want string }{
		{"label", "Production"},
		{"created_at", `{"gte":"2026-01-01T00:00:00Z"}`},
		{"last_used_at", `{"lt":"2026-02-01T00:00:00Z"}`},
		{"deactivated_at", `{"is_present":true}`},
		{"limit", "25"},
	} {
		if got := values.Get(tc.key); got != tc.want {
			t.Fatalf("expected %s=%q, got %q", tc.key, tc.want, got)
		}
	}
}

func TestListAPIKeysPaginates(t *testing.T) {
	pages := [][]byte{
		[]byte(`[{"token_last_4": "0001"}, {"token_last_4": "0002"}]`),
		[]byte(`[{"token_last_4": "0003"}]`),
	}

	var gotQueries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := pages[len(gotQueries)]
		gotQueries = append(gotQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", "3")
		w.Write(page)
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	apiKeys := collectAPIKeys(t, client.ListAPIKeys("inst_1", &APIKeyListParams{Limit: 2}))

	if len(apiKeys) != 3 {
		t.Fatalf("expected 3 api keys across both pages, got %d", len(apiKeys))
	}
	for i, want := range []string{"0001", "0002", "0003"} {
		if apiKeys[i].TokenLast4 != want {
			t.Fatalf("expected key %d to be %q, got %q", i, want, apiKeys[i].TokenLast4)
		}
	}
	if want := []string{"limit=2", "offset=2&limit=2"}; !reflect.DeepEqual(gotQueries, want) {
		t.Fatalf("expected queries %q, got %q", want, gotQueries)
	}
}

func TestListAPIKeysRequiresInstallationID(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if err := client.ListAPIKeys("", nil).Err(); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
}

func TestInstallationClientCreateAPIKey(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody APIKeyParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(apiKeyJSON))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	apiKey, err := client.CreateAPIKey("inst_1", &APIKeyParams{Label: "Production"})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/api_keys"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if want := "Production"; gotBody.Label != want {
		t.Fatalf("expected label %q in body, got %q", want, gotBody.Label)
	}

	// Create is the only response that carries the full token.
	if apiKey.ID != "sbx-ktesttoken000000000000000000000000" {
		t.Fatalf("unexpected id: %q", apiKey.ID)
	}
	if apiKey.Token != apiKey.ID {
		t.Fatalf("expected the full token to be returned, got %q", apiKey.Token)
	}
	if apiKey.Label == nil || *apiKey.Label != "Production" {
		t.Fatalf("unexpected label: %v", apiKey.Label)
	}
}

func TestInstallationClientUpdateAPIKey(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody APIKeyParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(apiKeyJSON))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	apiKey, err := client.UpdateAPIKey("inst_1", "sbx-ktesttoken000000000000000000000000", &APIKeyParams{Label: "Staging"})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/api_keys/sbx-ktesttoken000000000000000000000000"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "PATCH"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if want := "Staging"; gotBody.Label != want {
		t.Fatalf("expected label %q in body, got %q", want, gotBody.Label)
	}

	if apiKey.ID == "" {
		t.Fatal("expected the updated key to carry its id")
	}
}

func TestInstallationClientDeleteAPIKey(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		// Hint answers a successful delete with 204 and no body.
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	if err := client.DeleteAPIKey("inst_1", "sbx-ktesttoken000000000000000000000000"); err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/api_keys/sbx-ktesttoken000000000000000000000000"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "DELETE"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
}

func TestAPIKeyWritesRequireIDs(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	params := &APIKeyParams{Label: "Production"}

	if _, err := client.CreateAPIKey("", params); err == nil {
		t.Fatal("expected error for empty installation ID on create")
	}
	if _, err := client.UpdateAPIKey("", "key_1", params); err == nil {
		t.Fatal("expected error for empty installation ID on update")
	}
	if _, err := client.UpdateAPIKey("inst_1", "", params); err == nil {
		t.Fatal("expected error for empty api key ID on update")
	}
	if err := client.DeleteAPIKey("", "key_1"); err == nil {
		t.Fatal("expected error for empty installation ID on delete")
	}
	if err := client.DeleteAPIKey("inst_1", ""); err == nil {
		t.Fatal("expected error for empty api key ID on delete")
	}
}

func TestAPIKeyParamsValidate(t *testing.T) {
	if err := (&APIKeyParams{Label: "Production"}).Validate(); err != nil {
		t.Fatalf("expected valid params, got %v", err)
	}
	if err := (&APIKeyParams{}).Validate(); err == nil {
		t.Fatal("expected error for missing label")
	}
}
