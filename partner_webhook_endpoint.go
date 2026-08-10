package hint

import (
	"errors"
	"fmt"
)

// partnerWebhookEndpointsPath is the base path for the partner-level webhook
// endpoint endpoints, which manage delivery URLs for every integration
// connected to the partner rather than a single installation.
const partnerWebhookEndpointsPath = "/partner/webhook_endpoints"

// PartnerWebhookEndpointParams is the request body for creating or updating a
// partner-level webhook endpoint.
type PartnerWebhookEndpointParams struct {
	// WebhookURL is the HTTPS URL Hint delivers event payloads to. The endpoint
	// must respond with a 2xx within ~10s.
	WebhookURL string `json:"webhook_url,omitempty"`
	// Backend is the owning backend's public ID (PartnerBackend.ID). It is only
	// accepted on create; omit it to register the endpoint on the partner's
	// default backend.
	Backend string `json:"backend,omitempty"`
}

// Validate ensures the required fields for writing a partner-level webhook
// endpoint are present. The URL is optional on update as far as the API is
// concerned, but an update without it is a no-op, so it is required here for
// both operations.
func (p *PartnerWebhookEndpointParams) Validate() error {
	if p.WebhookURL == "" {
		return errors.New("webhook_url required")
	}
	return nil
}

// PartnerWebhookEndpointListParams are the filters accepted by
// GET /partner/webhook_endpoints. A nil *PartnerWebhookEndpointListParams
// applies no filters and uses the server's default page size.
type PartnerWebhookEndpointListParams struct {
	// WebhookURL narrows the results to endpoints with exactly this URL.
	WebhookURL string
	// Backend narrows the results to endpoints owned by a backend, identified by
	// its public ID (PartnerBackend.ID). Multiple IDs may be supplied
	// comma-separated.
	Backend string
	// CreatedAt filters on the endpoint's creation timestamp. Operands must be
	// ISO8601 timestamps, and the operators are limited to gt, gte, lt and lte.
	CreatedAt []*Operation
	// LastDeliveredAt filters on the endpoint's last delivery timestamp, in the
	// same form as CreatedAt.
	LastDeliveredAt []*Operation
	// Offset is the starting position for pagination. The iterator returned by
	// List advances it automatically when fetching subsequent pages.
	Offset uint64
	// Limit constrains the number of endpoints returned per page. Zero omits the
	// parameter so the server default applies.
	Limit uint64
}

// toListParams converts the typed filters into the generic ListParams the
// backend query encoder and the Iter pagination both operate on. A nil receiver
// yields empty (unfiltered) params.
func (p *PartnerWebhookEndpointListParams) toListParams() *ListParams {
	listParams := &ListParams{}
	if p == nil {
		return listParams
	}

	if p.WebhookURL != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("webhook_url", p.WebhookURL))
	}
	if p.Backend != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("backend", p.Backend))
	}
	if len(p.CreatedAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "created_at",
			Operations: p.CreatedAt,
		})
	}
	if len(p.LastDeliveredAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "last_delivered_at",
			Operations: p.LastDeliveredAt,
		})
	}
	listParams.Offset = p.Offset
	listParams.Limit = p.Limit

	return listParams
}

// Encode returns the query string for the params, in the format documented at
// https://developers.hint.com/reference/making-requests#advanced-querying.
func (p *PartnerWebhookEndpointListParams) Encode() (string, error) {
	return p.toListParams().Encode()
}

// PartnerWebhookEndpointClient exposes the partner-level webhook endpoint
// endpoints. Unlike the installation-scoped endpoints on InstallationClient,
// these register URLs that receive events for every integration connected to
// the partner.
type PartnerWebhookEndpointClient interface {
	// List returns an iterator that paginates through the partner's webhook
	// endpoints (GET /partner/webhook_endpoints). A nil params lists every
	// endpoint.
	List(params *PartnerWebhookEndpointListParams) *WebhookEndpointIter
	// Create registers a URL to receive events for every integration connected
	// to the partner and returns the created endpoint.
	Create(params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error)
	// Update points an existing webhook endpoint at a new URL and returns the
	// updated endpoint.
	Update(webhookEndpointID string, params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error)
	// Delete removes a webhook endpoint, so Hint stops delivering events to it.
	Delete(webhookEndpointID string) error
}

type partnerWebhookEndpointClient struct {
	B   Backend
	Key string
}

// NewPartnerWebhookEndpointClient returns an implementation of
// PartnerWebhookEndpointClient. Options may be supplied to customize the
// client, for example WithBaseURL(SandboxAPIURL) together with WithPartnerKey
// for a sandbox partner key. When no partner key is supplied the client falls
// back to the package-global Key at call time.
func NewPartnerWebhookEndpointClient(opts ...Option) PartnerWebhookEndpointClient {
	return getC(opts...).PartnerWebhookEndpoints
}

// resolveKey mirrors oauthClient.resolveKey: prefer the per-client key,
// otherwise fall back to the package-global Key at call time.
func (c partnerWebhookEndpointClient) resolveKey() string {
	if c.Key != "" {
		return c.Key
	}
	return Key
}

func (c partnerWebhookEndpointClient) List(params *PartnerWebhookEndpointListParams) *WebhookEndpointIter {
	iter := GetIter(params.toListParams(), func(lp *ListParams) ([]interface{}, ListMeta, error) {
		var meta ListMeta

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		path := partnerWebhookEndpointsPath
		if encodedParams != "" {
			path += "?" + encodedParams
		}

		var endpoints []*WebhookEndpoint
		resHeaders, err := c.B.Call("GET", path, c.resolveKey(), nil, &endpoints,
			WithHeader(hintVersionHeader, HintVersionMarketplace))
		if err != nil {
			return nil, meta, err
		}

		if meta, err = parseListMeta(resHeaders); err != nil {
			return nil, meta, err
		}

		ret := make([]interface{}, len(endpoints))
		for i, endpoint := range endpoints {
			ret[i] = endpoint
		}

		return ret, meta, nil
	})

	return &WebhookEndpointIter{Iter: iter}
}

func (c partnerWebhookEndpointClient) Create(params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error) {
	endpoint := &WebhookEndpoint{}
	if _, err := c.B.Call("POST", partnerWebhookEndpointsPath, c.resolveKey(), params, endpoint,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (c partnerWebhookEndpointClient) Update(webhookEndpointID string, params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error) {
	if webhookEndpointID == "" {
		return nil, errors.New("webhook_endpoint_id required")
	}

	endpoint := &WebhookEndpoint{}
	if _, err := c.B.Call("PATCH", fmt.Sprintf("%s/%s", partnerWebhookEndpointsPath, webhookEndpointID), c.resolveKey(), params, endpoint,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (c partnerWebhookEndpointClient) Delete(webhookEndpointID string) error {
	if webhookEndpointID == "" {
		return errors.New("webhook_endpoint_id required")
	}

	if _, err := c.B.Call("DELETE", fmt.Sprintf("%s/%s", partnerWebhookEndpointsPath, webhookEndpointID), c.resolveKey(), nil, nil,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return err
	}
	return nil
}
