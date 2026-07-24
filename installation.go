package hint

import (
	"errors"
	"fmt"
	"time"
)

// hintVersionHeader and HintVersionMarketplace are sent on the marketplace
// endpoints (installations + credential issuance) to opt in to that API version.
const (
	hintVersionHeader = "Hint-Version"

	// HintVersionMarketplace is the Hint-Version value that enables the
	// marketplace endpoints.
	HintVersionMarketplace = "marketplace_endpoints"
)

// Allowed values for Installation.Status.
const (
	InstallationStatusActive      = "active"
	InstallationStatusDeactivated = "deactivated"
)

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
type APIKey struct {
	ID            string     `json:"id"`
	Token         string     `json:"token"`
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

// InstallationClient exposes the marketplace endpoints for listing a partner's
// installations and pushing (or rotating) the credential a practice's custom
// apps use to call the partner API.
type InstallationClient interface {
	// List returns the partner's installations (GET /partner/installations), the
	// source of the installation IDs used by PushCredential.
	List() ([]*Installation, error)
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

func (c installationClient) List() ([]*Installation, error) {
	installations := []*Installation{}
	if _, err := c.B.Call("GET", "/partner/installations", c.resolveKey(), nil, &installations,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return installations, nil
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
