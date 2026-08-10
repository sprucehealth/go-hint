package hint

import (
	"errors"
	"fmt"
	"time"
)

// hintVersionHeader and HintVersionMarketplace are sent on the marketplace
// endpoints (installations, credential issuance, partner backends and
// partner-level webhook endpoints) to opt in to that API version.
const (
	hintVersionHeader = "Hint-Version"

	// HintVersionMarketplace is the Hint-Version value that enables the
	// marketplace endpoints.
	HintVersionMarketplace = "marketplace_endpoints"
)

// Allowed values for Installation.Status.
const (
	InstallationStatusOnboarding  = "onboarding"
	InstallationStatusPending     = "pending"
	InstallationStatusActive      = "active"
	InstallationStatusDeactivated = "deactivated"
)

// AvailableInstallationStatuses are the statuses an installation can be in, and
// the values accepted by InstallationListParams.Status.
var AvailableInstallationStatuses = []string{
	InstallationStatusOnboarding,
	InstallationStatusPending,
	InstallationStatusActive,
	InstallationStatusDeactivated,
}

// Installation represents a partner app installation on a practice, as returned
// by GET /partner/installations and POST /partner/installations/connect.
type Installation struct {
	ID                        string `json:"id"`
	AutoGrantPartnerAppAccess bool   `json:"auto_grant_partner_app_access"`
	// DefaultPartnerAppAdminRole / DefaultPartnerAppNonAdminRole are role
	// identifiers (e.g. "admin", "user") and are null when no default is set.
	DefaultPartnerAppAdminRole    *string   `json:"default_partner_app_admin_role"`
	DefaultPartnerAppNonAdminRole *string   `json:"default_partner_app_non_admin_role"`
	Status                        string    `json:"status"`
	Practice                      *Practice `json:"practice"`
	Product                       *Product  `json:"product"`
	// APIKeys is the practice's API credentials. It is returned by Connect (the
	// "installation object with API credentials"); the list endpoint omits it.
	APIKeys []*APIKey `json:"api_keys,omitempty"`
}

// Product represents the partner product an installation is for.
type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// APIKey is an API credential the practice's app uses to authenticate.
//
// Which fields are populated depends on the endpoint that returned the key.
// Connect and the single-key endpoints (InstallationClient.CreateAPIKey,
// UpdateAPIKey) return ID and Token; the list endpoint
// (InstallationClient.ListAPIKeys) returns neither, and identifies the key by
// TokenLast4 instead. Token itself only carries the full secret in the response
// to CreateAPIKey — Hint cannot return it again afterwards.
type APIKey struct {
	ID            string     `json:"id"`
	Token         string     `json:"token"`
	TokenLast4    string     `json:"token_last_4"`
	Label         *string    `json:"label"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	DeactivatedAt *time.Time `json:"deactivated_at"`
}

// CredentialParams is the request body for pushing a practice credential to an
// installation. See InstallationClient.PushCredential.
type CredentialParams struct {
	Credential CredentialInput `json:"credential"`
}

// CredentialInput is the credential pushed to Hint for an installation. Hint
// stores the payload encrypted and never returns it in any response.
type CredentialInput struct {
	// BaseURL is the base URL the practice's app should call the partner API at.
	BaseURL string `json:"base_url"`
	// Payload is the credential itself: an API key, token, or JSON blob, whatever
	// the partner API expects.
	Payload string `json:"payload"`
}

// Validate ensures the required fields for pushing a credential are present.
func (p *CredentialParams) Validate() error {
	if p.Credential.BaseURL == "" {
		return errors.New("base_url required")
	}
	if p.Credential.Payload == "" {
		return errors.New("payload required")
	}
	return nil
}

// Credential is the stored credential record returned by PushCredential. The
// payload is intentionally omitted: Hint stores it encrypted and never returns
// it in any response.
type Credential struct {
	ID            string     `json:"id"`
	BaseURL       string     `json:"base_url"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeactivatedAt *time.Time `json:"deactivated_at"`
}

// ConnectParams is the request body for connecting (activating) a pending
// installation. See InstallationClient.Connect.
type ConnectParams struct {
	// Code is the authorization code issued when the practice installed the
	// product.
	Code string `json:"code"`
	// Activate defaults to true on the server; set it to a pointer to false to
	// exchange the code without activating the installation. A nil pointer omits
	// the field so the server default applies.
	Activate *bool `json:"activate,omitempty"`
}

// Validate ensures the required fields for connecting an installation are present.
func (p *ConnectParams) Validate() error {
	if p.Code == "" {
		return errors.New("code required")
	}
	return nil
}

// InstallationListParams are the filters accepted by GET /partner/installations.
// A nil *InstallationListParams applies no filters and uses the server's default
// page size.
type InstallationListParams struct {
	// Status narrows the results to installations in this status. Use one of the
	// AvailableInstallationStatuses values.
	Status string
	// PartnerProduct narrows the results to a single partner product, identified
	// by its public product ID (Product.ID).
	PartnerProduct string
	// CreatedAt filters on the installation's creation timestamp. Operands must be
	// ISO8601 timestamps, and the operators are limited to gt, gte, lt and lte:
	// created_at accepts a range object rather than an exact match.
	CreatedAt []*Operation
	// Offset is the starting position for pagination. The iterator returned by
	// List advances it automatically when fetching subsequent pages.
	Offset uint64
	// Limit constrains the number of installations returned per page. Zero omits
	// the parameter so the server default applies.
	Limit uint64
}

// toListParams converts the typed filters into the generic ListParams the
// backend query encoder and the Iter pagination both operate on. A nil receiver
// yields empty (unfiltered) params.
func (p *InstallationListParams) toListParams() *ListParams {
	listParams := &ListParams{}
	if p == nil {
		return listParams
	}

	if p.Status != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("status", p.Status))
	}
	if p.PartnerProduct != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("partner_product", p.PartnerProduct))
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
func (p *InstallationListParams) Encode() (string, error) {
	return p.toListParams().Encode()
}

// InstallationIter paginates through a partner's installations. It behaves like
// Iter but exposes the current element as a typed *Installation.
type InstallationIter struct {
	*Iter
}

// Installation returns the installation the iterator currently points to.
func (it *InstallationIter) Installation() *Installation {
	installation, _ := it.Current().(*Installation)
	return installation
}

// InstallationClient exposes the marketplace endpoints for listing a partner's
// installations and pushing (or rotating) the credential a practice's custom
// apps use to call the partner API.
type InstallationClient interface {
	// List returns an iterator that paginates through the partner's installations
	// (GET /partner/installations), the source of the installation IDs used by
	// Get, Activate, Deactivate and PushCredential. A nil params lists every
	// installation.
	List(params *InstallationListParams) *InstallationIter
	// Get returns a single installation by ID (GET /partner/installations/{id}).
	// Deactivated installations are returned too, so a partner can inspect the
	// durable per-product install row across the install/uninstall cycle.
	Get(installationID string) (*Installation, error)
	// Activate activates a pending installation, once the practice has activated
	// the connection, and returns the updated installation object.
	Activate(installationID string) (*Installation, error)
	// PushCredential pushes (or rotates) the credential for the installation and
	// returns the stored record. Sending it again with a new base_url/payload
	// updates the single active credential rather than creating duplicates.
	PushCredential(installationID string, params *CredentialParams) (*Credential, error)
	// Connect activates a pending installation using the authorization code
	// issued when the practice installed the product, and returns the installation
	// object with its API credentials (Installation.APIKeys).
	Connect(params *ConnectParams) (*Installation, error)
	// Deactivate deactivates the installation and returns the updated installation
	// object. The practice's other installations are not affected.
	Deactivate(installationID string) (*Installation, error)
	// ListWebhookEndpoints returns an iterator that paginates through the URLs
	// Hint delivers the installation's webhook events to. A nil params lists every
	// endpoint.
	ListWebhookEndpoints(installationID string, params *WebhookEndpointListParams) *WebhookEndpointIter
	// CreateWebhookEndpoint registers a URL for the installation's webhook events
	// and returns the created endpoint.
	CreateWebhookEndpoint(installationID string, params *WebhookEndpointParams) (*WebhookEndpoint, error)
	// UpdateWebhookEndpoint points an existing webhook endpoint at a new URL and
	// returns the updated endpoint.
	UpdateWebhookEndpoint(installationID, webhookEndpointID string, params *WebhookEndpointParams) (*WebhookEndpoint, error)
	// DeleteWebhookEndpoint removes a webhook endpoint from the installation, so
	// Hint stops delivering the installation's events to it.
	DeleteWebhookEndpoint(installationID, webhookEndpointID string) error
	// ListAPIKeys returns an iterator that paginates through the API keys of the
	// installation's practice connection. A nil params lists every key. The listed
	// keys carry no ID or token, only their metadata.
	ListAPIKeys(installationID string, params *APIKeyListParams) *APIKeyIter
	// CreateAPIKey issues a new API key for the installation. The returned
	// APIKey.Token holds the full secret and is the only time Hint returns it, so
	// it has to be stored on receipt.
	CreateAPIKey(installationID string, params *APIKeyParams) (*APIKey, error)
	// UpdateAPIKey relabels an existing API key and returns the updated key.
	UpdateAPIKey(installationID, apiKeyID string, params *APIKeyParams) (*APIKey, error)
	// DeleteAPIKey removes an API key from the installation's practice
	// connection, so it can no longer be used to authenticate.
	DeleteAPIKey(installationID, apiKeyID string) error
}

type installationClient struct {
	B   Backend
	Key string
}

// NewInstallationClient returns an implementation of InstallationClient. Options
// may be supplied to customize the client, for example
// WithBaseURL(SandboxAPIURL) together with WithPartnerKey for a sandbox partner
// key. When no partner key is supplied the client falls back to the
// package-global Key at call time.
func NewInstallationClient(opts ...Option) InstallationClient {
	return getC(opts...).Installations
}

// resolveKey mirrors oauthClient.resolveKey: prefer the per-client key,
// otherwise fall back to the package-global Key at call time.
func (c installationClient) resolveKey() string {
	if c.Key != "" {
		return c.Key
	}
	return Key
}

func (c installationClient) List(params *InstallationListParams) *InstallationIter {
	iter := GetIter(params.toListParams(), func(lp *ListParams) ([]interface{}, ListMeta, error) {
		var meta ListMeta

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		path := "/partner/installations"
		if encodedParams != "" {
			path += "?" + encodedParams
		}

		var installations []*Installation
		resHeaders, err := c.B.Call("GET", path, c.resolveKey(), nil, &installations,
			WithHeader(hintVersionHeader, HintVersionMarketplace))
		if err != nil {
			return nil, meta, err
		}

		if meta, err = parseListMeta(resHeaders); err != nil {
			return nil, meta, err
		}

		ret := make([]interface{}, len(installations))
		for i, installation := range installations {
			ret[i] = installation
		}

		return ret, meta, nil
	})

	return &InstallationIter{Iter: iter}
}

func (c installationClient) Get(installationID string) (*Installation, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	installation := &Installation{}
	if _, err := c.B.Call("GET", fmt.Sprintf("/partner/installations/%s", installationID), c.resolveKey(), nil, installation,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return installation, nil
}

func (c installationClient) Activate(installationID string) (*Installation, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	installation := &Installation{}
	if _, err := c.B.Call("POST", fmt.Sprintf("/partner/installations/%s/activate", installationID), c.resolveKey(), nil, installation,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return installation, nil
}

func (c installationClient) PushCredential(installationID string, params *CredentialParams) (*Credential, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	credential := &Credential{}
	if _, err := c.B.Call("POST", fmt.Sprintf("/partner/installations/%s/credential", installationID), c.resolveKey(), params, credential,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return credential, nil
}

func (c installationClient) Connect(params *ConnectParams) (*Installation, error) {
	installation := &Installation{}
	if _, err := c.B.Call("POST", "/partner/installations/connect", c.resolveKey(), params, installation,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return installation, nil
}

func (c installationClient) Deactivate(installationID string) (*Installation, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	installation := &Installation{}
	if _, err := c.B.Call("POST", fmt.Sprintf("/partner/installations/%s/deactivate", installationID), c.resolveKey(), nil, installation,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return installation, nil
}
