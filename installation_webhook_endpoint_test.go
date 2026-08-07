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

const webhookEndpointJSON = `{
	"id": "whep-ab12C345DeF6",
	"created_at": "2026-07-16T20:15:25.529087Z",
	"updated_at": "2026-07-23T16:04:55.460804Z",
	"last_delivered_at": "2026-07-24T09:00:00.000000Z",
	"webhook_url": "https://hooks.your-service.com/hint",
	"partner_backend": {"id": "pbe-aaaa1111bbbb", "name": "Production"}
}`

// collectWebhookEndpoints drains a webhook endpoint iterator into a slice,
// failing the test if the iterator reports an error.
func collectWebhookEndpoints(t *testing.T, iter *WebhookEndpointIter) []*WebhookEndpoint {
	t.Helper()

	var endpoints []*WebhookEndpoint
	for iter.Next() {
		endpoints = append(endpoints, iter.WebhookEndpoint())
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}
	return endpoints
}

func TestWebhookEndpointListParamsEncode(t *testing.T) {
	cases := []struct {
		name     string
		params   *WebhookEndpointListParams
		expected string
	}{
		{"nil", nil, ""},
		{"empty", &WebhookEndpointListParams{}, ""},
		{
			"webhook url",
			&WebhookEndpointListParams{WebhookURL: "https://hooks.your-service.com/hint"},
			"webhook_url=https%3A%2F%2Fhooks.your-service.com%2Fhint",
		},
		{
			"last delivered at",
			&WebhookEndpointListParams{LastDeliveredAt: []*Operation{
				{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"},
			}},
			"last_delivered_at=%7B%22lt%22%3A%222026-02-01T00%3A00%3A00Z%22}",
		},
		{
			"all params",
			&WebhookEndpointListParams{
				WebhookURL:      "https://hooks.your-service.com/hint",
				CreatedAt:       []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
				LastDeliveredAt: []*Operation{{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"}},
				Offset:          10,
				Limit:           25,
			},
			"webhook_url=https%3A%2F%2Fhooks.your-service.com%2Fhint" +
				"&created_at=%7B%22gte%22%3A%222026-01-01T00%3A00%3A00Z%22}" +
				"&last_delivered_at=%7B%22lt%22%3A%222026-02-01T00%3A00%3A00Z%22}" +
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

func TestInstallationClientListWebhookEndpoints(t *testing.T) {
	var gotPath, gotQuery, gotMethod, gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotMethod = r.Method
		gotKey, _, _ = r.BasicAuth()
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[" + webhookEndpointJSON + "]"))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	endpoints := collectWebhookEndpoints(t, client.ListWebhookEndpoints("sbx-inst-aaaa1111bbbb", nil))

	if want := "/partner/installations/sbx-inst-aaaa1111bbbb/webhook_endpoints"; gotPath != want {
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

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 webhook endpoint, got %d", len(endpoints))
	}
	endpoint := endpoints[0]
	if endpoint.ID != "whep-ab12C345DeF6" {
		t.Fatalf("unexpected id: %q", endpoint.ID)
	}
	if endpoint.WebhookURL != "https://hooks.your-service.com/hint" {
		t.Fatalf("unexpected webhook_url: %q", endpoint.WebhookURL)
	}
	if endpoint.PartnerBackend == nil || endpoint.PartnerBackend.Name != "Production" {
		t.Fatalf("unexpected partner_backend: %+v", endpoint.PartnerBackend)
	}
	if endpoint.CreatedAt.IsZero() || endpoint.UpdatedAt.IsZero() {
		t.Fatalf("expected created_at/updated_at to be parsed, got %v / %v", endpoint.CreatedAt, endpoint.UpdatedAt)
	}
	if endpoint.LastDeliveredAt == nil || endpoint.LastDeliveredAt.IsZero() {
		t.Fatalf("expected last_delivered_at to be parsed, got %v", endpoint.LastDeliveredAt)
	}
}

func TestListWebhookEndpointsSendsFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	iter := client.ListWebhookEndpoints("inst_1", &WebhookEndpointListParams{
		WebhookURL:      "https://hooks.your-service.com/hint",
		CreatedAt:       []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
		LastDeliveredAt: []*Operation{{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"}},
		Limit:           25,
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
		{"webhook_url", "https://hooks.your-service.com/hint"},
		{"created_at", `{"gte":"2026-01-01T00:00:00Z"}`},
		{"last_delivered_at", `{"lt":"2026-02-01T00:00:00Z"}`},
		{"limit", "25"},
	} {
		if got := values.Get(tc.key); got != tc.want {
			t.Fatalf("expected %s=%q, got %q", tc.key, tc.want, got)
		}
	}
}

func TestListWebhookEndpointsPaginates(t *testing.T) {
	pages := [][]byte{
		[]byte(`[{"id": "whep_1"}, {"id": "whep_2"}]`),
		[]byte(`[{"id": "whep_3"}]`),
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
	endpoints := collectWebhookEndpoints(t, client.ListWebhookEndpoints("inst_1", &WebhookEndpointListParams{Limit: 2}))

	if len(endpoints) != 3 {
		t.Fatalf("expected 3 webhook endpoints across both pages, got %d", len(endpoints))
	}
	for i, want := range []string{"whep_1", "whep_2", "whep_3"} {
		if endpoints[i].ID != want {
			t.Fatalf("expected endpoint %d to be %q, got %q", i, want, endpoints[i].ID)
		}
	}
	if want := []string{"limit=2", "offset=2&limit=2"}; !reflect.DeepEqual(gotQueries, want) {
		t.Fatalf("expected queries %q, got %q", want, gotQueries)
	}
}

func TestListWebhookEndpointsRequiresInstallationID(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if err := client.ListWebhookEndpoints("", nil).Err(); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
}

func TestInstallationClientCreateWebhookEndpoint(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody WebhookEndpointParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(webhookEndpointJSON))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	endpoint, err := client.CreateWebhookEndpoint("inst_1", &WebhookEndpointParams{
		WebhookURL: "https://hooks.your-service.com/hint",
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/webhook_endpoints"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if want := "https://hooks.your-service.com/hint"; gotBody.WebhookURL != want {
		t.Fatalf("expected webhook_url %q in body, got %q", want, gotBody.WebhookURL)
	}

	if endpoint.ID != "whep-ab12C345DeF6" {
		t.Fatalf("unexpected id: %q", endpoint.ID)
	}
}

func TestInstallationClientUpdateWebhookEndpoint(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody WebhookEndpointParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(webhookEndpointJSON))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	endpoint, err := client.UpdateWebhookEndpoint("inst_1", "whep-ab12C345DeF6", &WebhookEndpointParams{
		WebhookURL: "https://hooks.your-service.com/hint-v2",
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/webhook_endpoints/whep-ab12C345DeF6"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "PATCH"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if want := "https://hooks.your-service.com/hint-v2"; gotBody.WebhookURL != want {
		t.Fatalf("expected webhook_url %q in body, got %q", want, gotBody.WebhookURL)
	}

	if endpoint.PartnerBackend == nil || endpoint.PartnerBackend.ID != "pbe-aaaa1111bbbb" {
		t.Fatalf("unexpected partner_backend: %+v", endpoint.PartnerBackend)
	}
}

func TestInstallationClientDeleteWebhookEndpoint(t *testing.T) {
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
	if err := client.DeleteWebhookEndpoint("inst_1", "whep-ab12C345DeF6"); err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/webhook_endpoints/whep-ab12C345DeF6"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "DELETE"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
}

func TestWebhookEndpointWritesRequireIDs(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	params := &WebhookEndpointParams{WebhookURL: "https://hooks.your-service.com/hint"}

	if _, err := client.CreateWebhookEndpoint("", params); err == nil {
		t.Fatal("expected error for empty installation ID on create")
	}
	if _, err := client.UpdateWebhookEndpoint("", "whep_1", params); err == nil {
		t.Fatal("expected error for empty installation ID on update")
	}
	if _, err := client.UpdateWebhookEndpoint("inst_1", "", params); err == nil {
		t.Fatal("expected error for empty webhook endpoint ID on update")
	}
	if err := client.DeleteWebhookEndpoint("", "whep_1"); err == nil {
		t.Fatal("expected error for empty installation ID on delete")
	}
	if err := client.DeleteWebhookEndpoint("inst_1", ""); err == nil {
		t.Fatal("expected error for empty webhook endpoint ID on delete")
	}
}

func TestWebhookEndpointParamsValidate(t *testing.T) {
	if err := (&WebhookEndpointParams{WebhookURL: "https://x"}).Validate(); err != nil {
		t.Fatalf("expected valid params, got %v", err)
	}
	if err := (&WebhookEndpointParams{}).Validate(); err == nil {
		t.Fatal("expected error for missing webhook_url")
	}
}
