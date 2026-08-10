package hint

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const partnerBackendJSON = `{
	"id": "pbnd-aaaa1111bbbb",
	"name": "Production",
	"activation_type": "instant",
	"auth_type": "automatic_redirect",
	"created_at": "2026-07-16T20:15:25.529087Z",
	"updated_at": "2026-07-23T16:04:55.460804Z",
	"listened_events": ["patient.created", "patient.updated"],
	"redirect_url": "https://your-service.com/oauth/callback",
	"localhost_redirect_url": "http://localhost:3000/oauth/callback",
	"localhost_webhook_url": null,
	"webhooks_disabled": false,
	"webhooks_paused": true,
	"unmute_self_webhooks": true,
	"max_requests_per_minute": 120,
	"api_credentials_configuration": {
		"id": "pacc-cccc2222dddd",
		"spec_url": "https://your-service.com/openapi.json",
		"docs_url": "https://your-service.com/docs",
		"issuance_mode": "push",
		"exchange_url": "https://your-service.com/oauth/token",
		"call_path": "proxy",
		"created_at": "2026-07-16T20:15:25.529087Z",
		"updated_at": "2026-07-23T16:04:55.460804Z"
	},
	"products": [{"id": "prod-eeee3333ffff", "name": "Scheduler", "slug": "scheduler"}],
	"webhooks_signature_api_key": {
		"id": "apik-gggg4444hhhh",
		"label": "webhook-signing",
		"created_at": "2026-07-16T20:15:25.529087Z",
		"last_used_at": null,
		"deactivated_at": null
	}
}`

// collectPartnerBackends drains a partner backend iterator into a slice,
// failing the test if the iterator reports an error.
func collectPartnerBackends(t *testing.T, iter *PartnerBackendIter) []*PartnerBackend {
	t.Helper()

	var backends []*PartnerBackend
	for iter.Next() {
		backends = append(backends, iter.PartnerBackend())
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}
	return backends
}

func TestPartnerBackendListParamsEncode(t *testing.T) {
	cases := []struct {
		name     string
		params   *PartnerBackendListParams
		expected string
	}{
		{"nil", nil, ""},
		{"empty", &PartnerBackendListParams{}, ""},
		{
			"created at",
			&PartnerBackendListParams{CreatedAt: []*Operation{
				{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"},
			}},
			"created_at=%7B%22gte%22%3A%222026-01-01T00%3A00%3A00Z%22}",
		},
		{
			"all params",
			&PartnerBackendListParams{
				CreatedAt: []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
				Offset:    10,
				Limit:     25,
			},
			"created_at=%7B%22gte%22%3A%222026-01-01T00%3A00%3A00Z%22}&offset=10&limit=25",
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

func TestPartnerBackendClientList(t *testing.T) {
	var gotPath, gotQuery, gotMethod, gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotMethod = r.Method
		gotKey, _, _ = r.BasicAuth()
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[" + partnerBackendJSON + "]"))
	}))
	defer srv.Close()

	client := NewPartnerBackendClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	backends := collectPartnerBackends(t, client.List(nil))

	if want := "/partner/backends"; gotPath != want {
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

	if len(backends) != 1 {
		t.Fatalf("expected 1 backend, got %d", len(backends))
	}
	backend := backends[0]
	if backend.ID != "pbnd-aaaa1111bbbb" {
		t.Fatalf("unexpected id: %q", backend.ID)
	}
	if backend.Name != "Production" {
		t.Fatalf("unexpected name: %q", backend.Name)
	}
	if backend.ActivationType != BackendActivationTypeInstant {
		t.Fatalf("unexpected activation_type: %q", backend.ActivationType)
	}
	if backend.AuthType != BackendAuthTypeAutomaticRedirect {
		t.Fatalf("unexpected auth_type: %q", backend.AuthType)
	}
	if want := []string{"patient.created", "patient.updated"}; !reflect.DeepEqual(backend.ListenedEvents, want) {
		t.Fatalf("expected listened_events %q, got %q", want, backend.ListenedEvents)
	}
	if backend.LocalhostWebhookURL != nil {
		t.Fatalf("expected null localhost_webhook_url, got %q", *backend.LocalhostWebhookURL)
	}
	if backend.WebhooksDisabled || !backend.WebhooksPaused || !backend.UnmuteSelfWebhooks {
		t.Fatalf("unexpected webhook flags: %+v", backend)
	}
	if backend.MaxRequestsPerMinute == nil || *backend.MaxRequestsPerMinute != 120 {
		t.Fatalf("unexpected max_requests_per_minute: %v", backend.MaxRequestsPerMinute)
	}
	if cfg := backend.APICredentialsConfiguration; cfg == nil ||
		cfg.ID != "pacc-cccc2222dddd" ||
		cfg.IssuanceMode != CredentialIssuanceModePush ||
		cfg.CallPath != CredentialCallPathProxy {
		t.Fatalf("unexpected api_credentials_configuration: %+v", cfg)
	}
	if len(backend.Products) != 1 || backend.Products[0].Slug != "scheduler" {
		t.Fatalf("unexpected products: %+v", backend.Products)
	}
	if key := backend.WebhooksSignatureAPIKey; key == nil || key.ID != "apik-gggg4444hhhh" {
		t.Fatalf("unexpected webhooks_signature_api_key: %+v", key)
	}
	if backend.CreatedAt.IsZero() || backend.UpdatedAt.IsZero() {
		t.Fatalf("expected created_at/updated_at to be parsed, got %v / %v", backend.CreatedAt, backend.UpdatedAt)
	}
}

func TestListPartnerBackendsPaginates(t *testing.T) {
	pages := [][]byte{
		[]byte(`[{"id": "pbnd_1"}, {"id": "pbnd_2"}]`),
		[]byte(`[{"id": "pbnd_3"}]`),
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

	client := NewPartnerBackendClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	backends := collectPartnerBackends(t, client.List(&PartnerBackendListParams{Limit: 2}))

	if len(backends) != 3 {
		t.Fatalf("expected 3 backends across both pages, got %d", len(backends))
	}
	for i, want := range []string{"pbnd_1", "pbnd_2", "pbnd_3"} {
		if backends[i].ID != want {
			t.Fatalf("expected backend %d to be %q, got %q", i, want, backends[i].ID)
		}
	}
	if want := []string{"limit=2", "offset=2&limit=2"}; !reflect.DeepEqual(gotQueries, want) {
		t.Fatalf("expected queries %q, got %q", want, gotQueries)
	}
}

func TestPartnerBackendClientGet(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(partnerBackendJSON))
	}))
	defer srv.Close()

	client := NewPartnerBackendClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	backend, err := client.Get("pbnd-aaaa1111bbbb")
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/backends/pbnd-aaaa1111bbbb"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "GET"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if backend.ID != "pbnd-aaaa1111bbbb" {
		t.Fatalf("unexpected id: %q", backend.ID)
	}
}

func TestPartnerBackendClientUpdate(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(partnerBackendJSON))
	}))
	defer srv.Close()

	paused := true
	maxRPM := int64(120)
	client := NewPartnerBackendClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	backend, err := client.Update("pbnd-aaaa1111bbbb", &PartnerBackendUpdateParams{
		Name:                    "Production",
		AuthType:                BackendAuthTypeAutomaticRedirect,
		WebhooksPaused:          &paused,
		MaxRequestsPerMinute:    &maxRPM,
		ListenedEvents:          []string{"patient.created"},
		WebhooksSignatureAPIKey: &APIKeyRef{ID: "apik-gggg4444hhhh"},
		APICredentialsConfiguration: &APICredentialsConfigurationParams{
			IssuanceMode: CredentialIssuanceModePush,
			CallPath:     CredentialCallPathProxy,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/backends/pbnd-aaaa1111bbbb"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "PATCH"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}

	if got := gotBody["name"]; got != "Production" {
		t.Fatalf("expected name %q in body, got %v", "Production", got)
	}
	if got := gotBody["auth_type"]; got != BackendAuthTypeAutomaticRedirect {
		t.Fatalf("expected auth_type %q in body, got %v", BackendAuthTypeAutomaticRedirect, got)
	}
	if got := gotBody["webhooks_paused"]; got != true {
		t.Fatalf("expected webhooks_paused true in body, got %v", got)
	}
	if got := gotBody["max_requests_per_minute"]; got != float64(120) {
		t.Fatalf("expected max_requests_per_minute 120 in body, got %v", got)
	}
	if got, ok := gotBody["webhooks_signature_api_key"].(map[string]interface{}); !ok || got["id"] != "apik-gggg4444hhhh" {
		t.Fatalf("expected webhooks_signature_api_key reference in body, got %v", gotBody["webhooks_signature_api_key"])
	}
	if got, ok := gotBody["api_credentials_configuration"].(map[string]interface{}); !ok || got["issuance_mode"] != CredentialIssuanceModePush {
		t.Fatalf("expected api_credentials_configuration in body, got %v", gotBody["api_credentials_configuration"])
	}
	// Unset optional fields must be omitted so Hint leaves them unchanged.
	for _, key := range []string{"activation_type", "redirect_url", "webhooks_disabled", "unmute_self_webhooks"} {
		if _, ok := gotBody[key]; ok {
			t.Fatalf("expected %s to be omitted from body, got %v", key, gotBody[key])
		}
	}

	if backend.ID != "pbnd-aaaa1111bbbb" {
		t.Fatalf("unexpected id: %q", backend.ID)
	}
}

func TestPartnerBackendReadsRequireID(t *testing.T) {
	client := NewPartnerBackendClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if _, err := client.Get(""); err == nil {
		t.Fatal("expected error for empty backend ID on get")
	}
	if _, err := client.Update("", &PartnerBackendUpdateParams{Name: "x"}); err == nil {
		t.Fatal("expected error for empty backend ID on update")
	}
}

func TestPartnerBackendUpdateParamsValidate(t *testing.T) {
	if err := (&PartnerBackendUpdateParams{}).Validate(); err != nil {
		t.Fatalf("expected empty params to be valid, got %v", err)
	}
	if err := (&PartnerBackendUpdateParams{WebhooksSignatureAPIKey: &APIKeyRef{}}).Validate(); err == nil {
		t.Fatal("expected error for webhooks_signature_api_key without id")
	}
	if err := (&PartnerBackendUpdateParams{WebhooksSignatureAPIKey: &APIKeyRef{ID: "apik_1"}}).Validate(); err != nil {
		t.Fatalf("expected valid params, got %v", err)
	}
}
