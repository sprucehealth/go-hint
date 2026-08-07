package hint

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// collect drains an installation iterator into a slice, failing the test if the
// iterator reports an error.
func collect(t *testing.T, iter *InstallationIter) []*Installation {
	t.Helper()

	var installations []*Installation
	for iter.Next() {
		installations = append(installations, iter.Installation())
	}
	if err := iter.Err(); err != nil {
		t.Fatal(err)
	}
	return installations
}

func TestInstallationClientList(t *testing.T) {
	var gotPath, gotQuery, gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotKey, _, _ = r.BasicAuth()
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": "sbx-inst-aaaa1111bbbb",
				"auto_grant_partner_app_access": true,
				"default_partner_app_admin_role": null,
				"default_partner_app_non_admin_role": null,
				"status": "active",
				"practice": {"id": "sbx-pra-cccc2222dddd", "name": "Test Practice One"},
				"product": {"id": "ppro-eeee3333ffff", "name": "Test Product One", "slug": "test-product-one"}
			},
			{
				"id": "sbx-inst-gggg4444hhhh",
				"auto_grant_partner_app_access": true,
				"status": "deactivated",
				"practice": {"id": "sbx-pra-iiii5555jjjj", "name": "Test Practice Two"},
				"product": {"id": "ppro-kkkk6666llll", "name": "Test Product Two", "slug": "test-product-two"}
			}
		]`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	installations := collect(t, client.List(nil))

	if want := "/partner/installations"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("expected no query string for nil params, got %q", gotQuery)
	}
	if want := "sandbox-key"; gotKey != want {
		t.Fatalf("expected partner key %q, got %q", want, gotKey)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}

	if len(installations) != 2 {
		t.Fatalf("expected 2 installations, got %d", len(installations))
	}

	first := installations[0]
	if first.ID != "sbx-inst-aaaa1111bbbb" {
		t.Fatalf("unexpected id: %q", first.ID)
	}
	if !first.AutoGrantPartnerAppAccess {
		t.Fatal("expected auto_grant_partner_app_access to be true")
	}
	if first.Status != InstallationStatusActive {
		t.Fatalf("expected status %q, got %q", InstallationStatusActive, first.Status)
	}
	if first.DefaultPartnerAppAdminRole != nil || first.DefaultPartnerAppNonAdminRole != nil {
		t.Fatalf("expected nil default roles, got %v / %v", first.DefaultPartnerAppAdminRole, first.DefaultPartnerAppNonAdminRole)
	}
	if first.Practice == nil || first.Practice.ID != "sbx-pra-cccc2222dddd" {
		t.Fatalf("unexpected practice: %+v", first.Practice)
	}
	if first.Product == nil || first.Product.Slug != "test-product-one" {
		t.Fatalf("unexpected product: %+v", first.Product)
	}
	if installations[1].Status != InstallationStatusDeactivated {
		t.Fatalf("expected status %q, got %q", InstallationStatusDeactivated, installations[1].Status)
	}
}

func TestInstallationListParamsEncode(t *testing.T) {
	cases := []struct {
		name     string
		params   *InstallationListParams
		expected string
	}{
		{"nil", nil, ""},
		{"empty", &InstallationListParams{}, ""},
		{"status", &InstallationListParams{Status: InstallationStatusActive}, "status=active"},
		{
			"partner product",
			&InstallationListParams{PartnerProduct: "ppro-eeee3333ffff"},
			"partner_product=ppro-eeee3333ffff",
		},
		{
			"created at range",
			&InstallationListParams{CreatedAt: []*Operation{
				{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"},
				{Operator: OperatorLessThan, Operand: "2026-02-01T00:00:00Z"},
			}},
			"created_at=%7B%22gte%22%3A%222026-01-01T00%3A00%3A00Z%22%2C%22lt%22%3A%222026-02-01T00%3A00%3A00Z%22}",
		},
		{"pagination", &InstallationListParams{Offset: 10, Limit: 25}, "offset=10&limit=25"},
		{
			"all params",
			&InstallationListParams{
				Status:         InstallationStatusPending,
				PartnerProduct: "ppro-eeee3333ffff",
				CreatedAt:      []*Operation{{Operator: OperatorGreaterThan, Operand: "2026-01-01T00:00:00Z"}},
				Offset:         10,
				Limit:          25,
			},
			"status=pending&partner_product=ppro-eeee3333ffff&created_at=%7B%22gt%22%3A%222026-01-01T00%3A00%3A00Z%22}&offset=10&limit=25",
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

func TestInstallationClientListSendsFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	iter := client.List(&InstallationListParams{
		Status:         InstallationStatusActive,
		PartnerProduct: "ppro-eeee3333ffff",
		CreatedAt:      []*Operation{{Operator: OperatorGreaterThanEqualTo, Operand: "2026-01-01T00:00:00Z"}},
		Limit:          25,
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
		{"status", "active"},
		{"partner_product", "ppro-eeee3333ffff"},
		{"created_at", `{"gte":"2026-01-01T00:00:00Z"}`},
		{"limit", "25"},
	} {
		if got := values.Get(tc.key); got != tc.want {
			t.Fatalf("expected %s=%q, got %q", tc.key, tc.want, got)
		}
	}
	if _, ok := values["offset"]; ok {
		t.Fatalf("expected offset to be omitted when zero, got %q", gotQuery)
	}
}

func TestInstallationClientListPaginates(t *testing.T) {
	pages := [][]byte{
		[]byte(`[{"id": "inst_1"}, {"id": "inst_2"}]`),
		[]byte(`[{"id": "inst_3"}]`),
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
	iter := client.List(&InstallationListParams{Limit: 2})
	installations := collect(t, iter)

	if len(installations) != 3 {
		t.Fatalf("expected 3 installations across both pages, got %d", len(installations))
	}
	for i, want := range []string{"inst_1", "inst_2", "inst_3"} {
		if installations[i].ID != want {
			t.Fatalf("expected installation %d to be %q, got %q", i, want, installations[i].ID)
		}
	}
	if want := []string{"limit=2", "offset=2&limit=2"}; !reflect.DeepEqual(gotQueries, want) {
		t.Fatalf("expected queries %q, got %q", want, gotQueries)
	}
	if iter.Count() != 3 {
		t.Fatalf("expected total count 3, got %d", iter.Count())
	}
	if iter.HasMore() {
		t.Fatal("expected HasMore to be false after draining the iterator")
	}
}

func TestInstallationClientListResumesFromOffset(t *testing.T) {
	pages := [][]byte{
		[]byte(`[{"id": "inst_2"}, {"id": "inst_3"}]`),
		[]byte(`[{"id": "inst_4"}]`),
	}

	var gotQueries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := pages[len(gotQueries)]
		gotQueries = append(gotQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", "4")
		w.Write(page)
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	// Starting at offset 1 of 4 records: the second page must resume at 3, not 2.
	installations := collect(t, client.List(&InstallationListParams{Offset: 1, Limit: 2}))

	if len(installations) != 3 {
		t.Fatalf("expected 3 installations, got %d", len(installations))
	}
	if want := []string{"offset=1&limit=2", "offset=3&limit=2"}; !reflect.DeepEqual(gotQueries, want) {
		t.Fatalf("expected queries %q, got %q", want, gotQueries)
	}
}

func TestInstallationClientListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "unauthorized"}}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("bad-key"))
	iter := client.List(nil)
	if iter.Next() {
		t.Fatal("expected no installations on an error response")
	}
	if iter.Err() == nil {
		t.Fatal("expected the iterator to surface the API error")
	}
}

func TestInstallationClientGet(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "sbx-inst-aaaa1111bbbb",
			"auto_grant_partner_app_access": true,
			"default_partner_app_admin_role": "admin",
			"default_partner_app_non_admin_role": "user",
			"status": "active",
			"practice": {"id": "sbx-pra-cccc2222dddd", "name": "Test Practice One"},
			"product": {"id": "ppro-eeee3333ffff", "name": "Test Product One", "slug": "test-product-one"}
		}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	installation, err := client.Get("sbx-inst-aaaa1111bbbb")
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/sbx-inst-aaaa1111bbbb"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "GET"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}

	if installation.ID != "sbx-inst-aaaa1111bbbb" {
		t.Fatalf("unexpected installation id: %q", installation.ID)
	}
	if installation.Status != InstallationStatusActive {
		t.Fatalf("unexpected status: %q", installation.Status)
	}
	if installation.Practice == nil || installation.Practice.Name != "Test Practice One" {
		t.Fatalf("unexpected practice: %+v", installation.Practice)
	}
	if installation.Product == nil || installation.Product.Slug != "test-product-one" {
		t.Fatalf("unexpected product: %+v", installation.Product)
	}
	if installation.DefaultPartnerAppAdminRole == nil || *installation.DefaultPartnerAppAdminRole != "admin" {
		t.Fatalf("unexpected admin role: %v", installation.DefaultPartnerAppAdminRole)
	}
}

func TestInstallationClientActivate(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		gotBody, _ = ioutil.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "sbx-inst-aaaa1111bbbb",
			"auto_grant_partner_app_access": true,
			"default_partner_app_admin_role": null,
			"default_partner_app_non_admin_role": null,
			"status": "active",
			"practice": {"id": "sbx-pra-cccc2222dddd", "name": "Test Practice One"},
			"product": {"id": "ppro-eeee3333ffff", "name": "Test Product One", "slug": "test-product-one"}
		}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	installation, err := client.Activate("sbx-inst-aaaa1111bbbb")
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/sbx-inst-aaaa1111bbbb/activate"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if len(gotBody) != 0 {
		t.Fatalf("expected no request body, got %s", gotBody)
	}

	if installation.Status != InstallationStatusActive {
		t.Fatalf("expected status %q, got %q", InstallationStatusActive, installation.Status)
	}
}

func TestGetAndActivateRequireInstallationID(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if _, err := client.Get(""); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
	if _, err := client.Activate(""); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
}

func TestInstallationClientPushCredential(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody CredentialParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "sbx-ppcred-testtesttest",
			"base_url": "https://api.your-service.com",
			"created_at": "2026-07-23T16:04:55.460804Z",
			"deactivated_at": null,
			"updated_at": "2026-07-23T16:04:55.460804Z"
		}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	credential, err := client.PushCredential("inst_1", &CredentialParams{
		Credential: CredentialInput{
			BaseURL: "https://api.your-service.com",
			Payload: "the-secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/inst_1/credential"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if gotBody.Credential.BaseURL != "https://api.your-service.com" || gotBody.Credential.Payload != "the-secret" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}

	if credential.ID != "sbx-ppcred-testtesttest" {
		t.Fatalf("unexpected credential id: %q", credential.ID)
	}
	if credential.BaseURL != "https://api.your-service.com" {
		t.Fatalf("unexpected base_url: %q", credential.BaseURL)
	}
	if credential.CreatedAt.IsZero() || credential.UpdatedAt.IsZero() {
		t.Fatalf("expected created_at/updated_at to be parsed, got %v / %v", credential.CreatedAt, credential.UpdatedAt)
	}
	if credential.DeactivatedAt != nil {
		t.Fatalf("expected deactivated_at to be nil, got %v", credential.DeactivatedAt)
	}
}

func TestPushCredentialRequiresInstallationID(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if _, err := client.PushCredential("", &CredentialParams{
		Credential: CredentialInput{BaseURL: "https://x", Payload: "y"},
	}); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
}

func TestCredentialParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  CredentialParams
		wantErr bool
	}{
		{"valid", CredentialParams{CredentialInput{BaseURL: "https://x", Payload: "y"}}, false},
		{"missing base_url", CredentialParams{CredentialInput{Payload: "y"}}, true},
		{"missing payload", CredentialParams{CredentialInput{BaseURL: "https://x"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.params
			if err := p.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestInstallationClientConnect(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		body, _ := ioutil.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "sbx-inst-aaaa1111bbbb",
			"auto_grant_partner_app_access": true,
			"default_partner_app_admin_role": null,
			"default_partner_app_non_admin_role": null,
			"status": "active",
			"api_keys": [
				{
					"id": "sbx-ktesttoken000000000000000000000000",
					"created_at": "2026-07-16T20:15:25.529087Z",
					"deactivated_at": null,
					"label": null,
					"last_used_at": "2026-07-20T15:39:20.000000Z",
					"token": "sbx-ktesttoken000000000000000000000000"
				}
			],
			"practice": {"id": "sbx-pra-cccc2222dddd", "name": "Test Practice One"},
			"product": {"id": "ppro-eeee3333ffff", "name": "Test Product One", "slug": "test-product-one"}
		}`))
	}))
	defer srv.Close()

	activate := false
	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	installation, err := client.Connect(&ConnectParams{Code: "auth-code-123", Activate: &activate})
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/connect"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if gotBody["code"] != "auth-code-123" {
		t.Fatalf("unexpected code in body: %v", gotBody["code"])
	}
	if gotBody["activate"] != false {
		t.Fatalf("expected activate=false in body, got %v", gotBody["activate"])
	}

	if installation.ID != "sbx-inst-aaaa1111bbbb" {
		t.Fatalf("unexpected installation id: %q", installation.ID)
	}
	if installation.Status != InstallationStatusActive {
		t.Fatalf("unexpected status: %q", installation.Status)
	}
	if len(installation.APIKeys) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(installation.APIKeys))
	}
	key := installation.APIKeys[0]
	if key.ID != "sbx-ktesttoken000000000000000000000000" || key.Token != key.ID {
		t.Fatalf("unexpected api key id/token: %+v", key)
	}
	if key.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be parsed")
	}
	if key.LastUsedAt == nil || key.LastUsedAt.IsZero() {
		t.Fatalf("expected last_used_at to be parsed, got %v", key.LastUsedAt)
	}
	if key.DeactivatedAt != nil {
		t.Fatalf("expected deactivated_at to be nil, got %v", key.DeactivatedAt)
	}
	if key.Label != nil {
		t.Fatalf("expected label to be nil, got %v", key.Label)
	}
}

func TestConnectOmitsActivateWhenNil(t *testing.T) {
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = ioutil.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("k"))
	// A nil Activate must be omitted so the server default (true) applies.
	if _, err := client.Connect(&ConnectParams{Code: "auth-code-123"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawBody), "activate") {
		t.Fatalf("expected activate to be omitted when nil, body was: %s", rawBody)
	}
}

func TestConnectParamsValidate(t *testing.T) {
	if err := (&ConnectParams{Code: "x"}).Validate(); err != nil {
		t.Fatalf("expected valid params, got %v", err)
	}
	if err := (&ConnectParams{}).Validate(); err == nil {
		t.Fatal("expected error for missing code")
	}
}

func TestInstallationClientDeactivate(t *testing.T) {
	var gotPath, gotMethod, gotVersion string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotVersion = r.Header.Get("Hint-Version")
		gotBody, _ = ioutil.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "sbx-inst-aaaa1111bbbb",
			"auto_grant_partner_app_access": true,
			"default_partner_app_admin_role": "admin",
			"default_partner_app_non_admin_role": "user",
			"status": "deactivated",
			"practice": {"id": "sbx-pra-cccc2222dddd", "name": "Test Practice One"},
			"product": {"id": "ppro-eeee3333ffff", "name": "Test Product One", "slug": "test-product-one"}
		}`))
	}))
	defer srv.Close()

	client := NewInstallationClient(WithBaseURL(srv.URL), WithPartnerKey("sandbox-key"))
	installation, err := client.Deactivate("sbx-inst-aaaa1111bbbb")
	if err != nil {
		t.Fatal(err)
	}

	if want := "/partner/installations/sbx-inst-aaaa1111bbbb/deactivate"; gotPath != want {
		t.Fatalf("expected path %q, got %q", want, gotPath)
	}
	if want := "POST"; gotMethod != want {
		t.Fatalf("expected method %q, got %q", want, gotMethod)
	}
	if want := HintVersionMarketplace; gotVersion != want {
		t.Fatalf("expected Hint-Version %q, got %q", want, gotVersion)
	}
	if len(gotBody) != 0 {
		t.Fatalf("expected no request body, got %s", gotBody)
	}

	if installation.Status != InstallationStatusDeactivated {
		t.Fatalf("expected status %q, got %q", InstallationStatusDeactivated, installation.Status)
	}
	if installation.DefaultPartnerAppAdminRole == nil || *installation.DefaultPartnerAppAdminRole != "admin" {
		t.Fatalf("unexpected admin role: %v", installation.DefaultPartnerAppAdminRole)
	}
	if installation.DefaultPartnerAppNonAdminRole == nil || *installation.DefaultPartnerAppNonAdminRole != "user" {
		t.Fatalf("unexpected non-admin role: %v", installation.DefaultPartnerAppNonAdminRole)
	}
	if installation.Product == nil || installation.Product.Slug != "test-product-one" {
		t.Fatalf("unexpected product: %+v", installation.Product)
	}
}

func TestDeactivateRequiresInstallationID(t *testing.T) {
	client := NewInstallationClient(WithBaseURL("https://example.invalid"), WithPartnerKey("k"))
	if _, err := client.Deactivate(""); err == nil {
		t.Fatal("expected error for empty installation ID")
	}
}

func TestInstallationClientFallsBackToGlobalKey(t *testing.T) {
	prevKey := Key
	defer func() { Key = prevKey }()
	Key = "global-key"

	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, _, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	// No WithPartnerKey: the client should authenticate with the global Key.
	client := NewInstallationClient(WithBaseURL(srv.URL))
	if err := client.List(nil).Err(); err != nil {
		t.Fatal(err)
	}
	if want := "global-key"; gotKey != want {
		t.Fatalf("expected key %q, got %q", want, gotKey)
	}
}
