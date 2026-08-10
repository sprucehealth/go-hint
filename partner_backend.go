package hint

import (
	"errors"
	"fmt"
	"time"
)

// Allowed values for PartnerBackend.AuthType.
const (
	BackendAuthTypeManual            = "manual"
	BackendAuthTypeAutomaticRedirect = "automatic_redirect"
	BackendAuthTypeAutomaticHeadless = "automatic_headless"
)

// Allowed values for PartnerBackend.ActivationType.
const (
	BackendActivationTypeInstant          = "instant"
	BackendActivationTypePartnerActivate  = "partner_activate"
	BackendActivationTypePracticeActivate = "practice_activate"
)

// Allowed values for APICredentialsConfiguration.IssuanceMode.
const (
	CredentialIssuanceModePush = "push"
	CredentialIssuanceModePull = "pull"
)

// Allowed values for APICredentialsConfiguration.CallPath.
const (
	CredentialCallPathProxy  = "proxy"
	CredentialCallPathDirect = "direct"
)

// PartnerBackend is a partner backend as returned by the backend endpoints
// (GET /partner/backends). Each backend owns its own webhook delivery, OAuth
// config, and product catalog. Webhook endpoint responses embed an abbreviated
// form carrying only ID and Name.
type PartnerBackend struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ActivationType is one of the BackendActivationType constants.
	ActivationType string `json:"activation_type"`
	// AuthType is one of the BackendAuthType constants.
	AuthType string `json:"auth_type"`
	// ListenedEvents are the event types delivered to the backend's endpoints,
	// as resource.action pairs (e.g. "patient.created").
	ListenedEvents []string `json:"listened_events"`
	// RedirectURL is the OAuth redirect URL invoked after authorization_code
	// issuance.
	RedirectURL string `json:"redirect_url"`
	// LocalhostRedirectURL / LocalhostWebhookURL are the localhost URLs used in
	// place of the real ones when the requesting browser session has localhost
	// mode enabled. Sandbox partners only.
	LocalhostRedirectURL string  `json:"localhost_redirect_url"`
	LocalhostWebhookURL  *string `json:"localhost_webhook_url"`
	WebhooksDisabled     bool    `json:"webhooks_disabled"`
	WebhooksPaused       bool    `json:"webhooks_paused"`
	// UnmuteSelfWebhooks delivers webhooks for events triggered by the backend's
	// own requests.
	UnmuteSelfWebhooks bool `json:"unmute_self_webhooks"`
	// MaxRequestsPerMinute is the webhook delivery rate ceiling for the backend.
	// It is null when no ceiling is configured.
	MaxRequestsPerMinute        *int64                       `json:"max_requests_per_minute"`
	APICredentialsConfiguration *APICredentialsConfiguration `json:"api_credentials_configuration"`
	// Products are the partner products served by the backend.
	Products []*Product `json:"products"`
	// WebhooksSignatureAPIKey is the API key Hint signs the backend's webhook
	// deliveries with. Only the key's metadata is returned, never the secret.
	WebhooksSignatureAPIKey *APIKey   `json:"webhooks_signature_api_key"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// APICredentialsConfiguration describes how Hint issues or exchanges practice
// API credentials for a backend.
type APICredentialsConfiguration struct {
	ID      string `json:"id"`
	SpecURL string `json:"spec_url"`
	DocsURL string `json:"docs_url"`
	// IssuanceMode is one of the CredentialIssuanceMode constants.
	IssuanceMode string `json:"issuance_mode"`
	ExchangeURL  string `json:"exchange_url"`
	// CallPath is one of the CredentialCallPath constants.
	CallPath  string    `json:"call_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIKeyRef references an existing API key by ID, the shape the backend update
// endpoint accepts for webhooks_signature_api_key.
type APIKeyRef struct {
	ID string `json:"id"`
}

// APICredentialsConfigurationParams is the api_credentials_configuration
// portion of a backend update. All fields are optional; empty fields are
// omitted from the request.
type APICredentialsConfigurationParams struct {
	SpecURL string `json:"spec_url,omitempty"`
	DocsURL string `json:"docs_url,omitempty"`
	// IssuanceMode is one of the CredentialIssuanceMode constants.
	IssuanceMode string `json:"issuance_mode,omitempty"`
	ExchangeURL  string `json:"exchange_url,omitempty"`
	// CallPath is one of the CredentialCallPath constants.
	CallPath string `json:"call_path,omitempty"`
}

// PartnerBackendUpdateParams is the request body for updating a partner
// backend (PATCH /partner/backends/{id}). All fields are optional; empty
// fields are omitted from the request and left unchanged by Hint.
type PartnerBackendUpdateParams struct {
	// Name is a human-readable label for the backend.
	Name string `json:"name,omitempty"`
	// AuthType is one of the BackendAuthType constants.
	AuthType string `json:"auth_type,omitempty"`
	// ActivationType is one of the BackendActivationType constants.
	ActivationType string `json:"activation_type,omitempty"`
	// RedirectURL is the OAuth redirect URL invoked after authorization_code
	// issuance.
	RedirectURL string `json:"redirect_url,omitempty"`
	// LocalhostRedirectURL / LocalhostWebhookURL are the localhost URLs used in
	// place of the real ones when the requesting browser session has localhost
	// mode enabled. Sandbox partners only.
	LocalhostRedirectURL string `json:"localhost_redirect_url,omitempty"`
	LocalhostWebhookURL  string `json:"localhost_webhook_url,omitempty"`
	// WebhooksDisabled disables webhook delivery for the backend's endpoints. A
	// nil pointer omits the field so the current value is kept.
	WebhooksDisabled *bool `json:"webhooks_disabled,omitempty"`
	// WebhooksPaused pauses webhook delivery for the backend's endpoints.
	WebhooksPaused *bool `json:"webhooks_paused,omitempty"`
	// UnmuteSelfWebhooks delivers webhooks for events triggered by the backend's
	// own requests.
	UnmuteSelfWebhooks *bool `json:"unmute_self_webhooks,omitempty"`
	// MaxRequestsPerMinute is the webhook delivery rate ceiling for the backend.
	MaxRequestsPerMinute *int64 `json:"max_requests_per_minute,omitempty"`
	// ListenedEvents are the event types delivered to the backend's endpoints.
	ListenedEvents []string `json:"listened_events,omitempty"`
	// WebhooksSignatureAPIKey selects the API key Hint signs the backend's
	// webhook deliveries with, referenced by ID.
	WebhooksSignatureAPIKey     *APIKeyRef                         `json:"webhooks_signature_api_key,omitempty"`
	APICredentialsConfiguration *APICredentialsConfigurationParams `json:"api_credentials_configuration,omitempty"`
}

// Validate ensures the update params are well formed. Every field is optional,
// but a webhooks_signature_api_key reference must carry the key's ID.
func (p *PartnerBackendUpdateParams) Validate() error {
	if p.WebhooksSignatureAPIKey != nil && p.WebhooksSignatureAPIKey.ID == "" {
		return errors.New("webhooks_signature_api_key id required")
	}
	return nil
}

// PartnerBackendListParams are the filters accepted by GET /partner/backends.
// A nil *PartnerBackendListParams applies no filters and uses the server's
// default page size.
type PartnerBackendListParams struct {
	// CreatedAt filters on the backend's creation timestamp. Operands must be
	// ISO8601 timestamps, and the operators are limited to gt, gte, lt and lte.
	CreatedAt []*Operation
	// Offset is the starting position for pagination. The iterator returned by
	// List advances it automatically when fetching subsequent pages.
	Offset uint64
	// Limit constrains the number of backends returned per page. Zero omits the
	// parameter so the server default applies.
	Limit uint64
}

// toListParams converts the typed filters into the generic ListParams the
// backend query encoder and the Iter pagination both operate on. A nil receiver
// yields empty (unfiltered) params.
func (p *PartnerBackendListParams) toListParams() *ListParams {
	listParams := &ListParams{}
	if p == nil {
		return listParams
	}

	if len(p.CreatedAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "created_at",
			Operations: p.CreatedAt,
		})
	}
	listParams.Offset = p.Offset
	listParams.Limit = p.Limit

	return listParams
}

// Encode returns the query string for the params, in the format documented at
// https://developers.hint.com/reference/making-requests#advanced-querying.
func (p *PartnerBackendListParams) Encode() (string, error) {
	return p.toListParams().Encode()
}

// PartnerBackendIter paginates through a partner's backends. It behaves like
// Iter but exposes the current element as a typed *PartnerBackend.
type PartnerBackendIter struct {
	*Iter
}

// PartnerBackend returns the backend the iterator currently points to.
func (it *PartnerBackendIter) PartnerBackend() *PartnerBackend {
	backend, _ := it.Current().(*PartnerBackend)
	return backend
}

// PartnerBackendClient exposes the partner backend endpoints. Each backend
// owns its own webhook delivery, OAuth config, and product catalog.
type PartnerBackendClient interface {
	// List returns an iterator that paginates through the partner's backends
	// (GET /partner/backends). A nil params lists every backend.
	List(params *PartnerBackendListParams) *PartnerBackendIter
	// Get returns a single backend by ID (GET /partner/backends/{id}).
	Get(backendID string) (*PartnerBackend, error)
	// Update updates a backend's configuration and returns the updated backend
	// (PATCH /partner/backends/{id}).
	Update(backendID string, params *PartnerBackendUpdateParams) (*PartnerBackend, error)
}

type partnerBackendClient struct {
	B   Backend
	Key string
}

// NewPartnerBackendClient returns an implementation of PartnerBackendClient.
// Options may be supplied to customize the client, for example
// WithBaseURL(SandboxAPIURL) together with WithPartnerKey for a sandbox partner
// key. When no partner key is supplied the client falls back to the
// package-global Key at call time.
func NewPartnerBackendClient(opts ...Option) PartnerBackendClient {
	return getC(opts...).PartnerBackends
}

// resolveKey mirrors oauthClient.resolveKey: prefer the per-client key,
// otherwise fall back to the package-global Key at call time.
func (c partnerBackendClient) resolveKey() string {
	if c.Key != "" {
		return c.Key
	}
	return Key
}

func (c partnerBackendClient) List(params *PartnerBackendListParams) *PartnerBackendIter {
	iter := GetIter(params.toListParams(), func(lp *ListParams) ([]interface{}, ListMeta, error) {
		var meta ListMeta

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		path := "/partner/backends"
		if encodedParams != "" {
			path += "?" + encodedParams
		}

		var backends []*PartnerBackend
		resHeaders, err := c.B.Call("GET", path, c.resolveKey(), nil, &backends,
			WithHeader(hintVersionHeader, HintVersionMarketplace))
		if err != nil {
			return nil, meta, err
		}

		if meta, err = parseListMeta(resHeaders); err != nil {
			return nil, meta, err
		}

		ret := make([]interface{}, len(backends))
		for i, backend := range backends {
			ret[i] = backend
		}

		return ret, meta, nil
	})

	return &PartnerBackendIter{Iter: iter}
}

func (c partnerBackendClient) Get(backendID string) (*PartnerBackend, error) {
	if backendID == "" {
		return nil, errors.New("backend_id required")
	}

	backend := &PartnerBackend{}
	if _, err := c.B.Call("GET", fmt.Sprintf("/partner/backends/%s", backendID), c.resolveKey(), nil, backend,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return backend, nil
}

func (c partnerBackendClient) Update(backendID string, params *PartnerBackendUpdateParams) (*PartnerBackend, error) {
	if backendID == "" {
		return nil, errors.New("backend_id required")
	}

	backend := &PartnerBackend{}
	if _, err := c.B.Call("PATCH", fmt.Sprintf("/partner/backends/%s", backendID), c.resolveKey(), params, backend,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return backend, nil
}
