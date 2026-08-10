package hint

import (
	"errors"
	"fmt"
	"time"
)

// WebhookEndpoint is a URL that Hint delivers webhook events to, either for a
// single installation or (for the partner-level endpoints) for every
// integration connected to the partner.
type WebhookEndpoint struct {
	ID         string `json:"id"`
	WebhookURL string `json:"webhook_url"`
	// PartnerBackend is the partner backend the endpoint belongs to. Only its ID
	// and Name are populated here; see PartnerBackendClient for the full object.
	PartnerBackend *PartnerBackend `json:"partner_backend"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	// LastDeliveredAt is null until Hint has delivered an event to the endpoint.
	LastDeliveredAt *time.Time `json:"last_delivered_at"`
}

// WebhookEndpointParams is the request body for creating or updating an
// installation's webhook endpoint.
type WebhookEndpointParams struct {
	// WebhookURL is the URL Hint delivers the installation's events to.
	WebhookURL string `json:"webhook_url,omitempty"`
}

// Validate ensures the required fields for writing a webhook endpoint are
// present. The URL is optional on update as far as the API is concerned, but an
// update without it is a no-op, so it is required here for both operations.
func (p *WebhookEndpointParams) Validate() error {
	if p.WebhookURL == "" {
		return errors.New("webhook_url required")
	}
	return nil
}

// WebhookEndpointListParams are the filters accepted by
// GET /partner/installations/{installation_id}/webhook_endpoints. A nil
// *WebhookEndpointListParams applies no filters and uses the server's default
// page size.
type WebhookEndpointListParams struct {
	// WebhookURL narrows the results to endpoints with exactly this URL.
	WebhookURL string
	// CreatedAt filters on the endpoint's creation timestamp. Operands must be
	// ISO8601 timestamps, and the operators are limited to gt, gte, lt and lte.
	CreatedAt []*Operation
	// LastDeliveredAt filters on the endpoint's last delivery timestamp, in the
	// same form as CreatedAt.
	LastDeliveredAt []*Operation
	// Offset is the starting position for pagination. The iterator returned by
	// ListWebhookEndpoints advances it automatically when fetching subsequent
	// pages.
	Offset uint64
	// Limit constrains the number of endpoints returned per page. Zero omits the
	// parameter so the server default applies.
	Limit uint64
}

// toListParams converts the typed filters into the generic ListParams the
// backend query encoder and the Iter pagination both operate on. A nil receiver
// yields empty (unfiltered) params.
func (p *WebhookEndpointListParams) toListParams() *ListParams {
	listParams := &ListParams{}
	if p == nil {
		return listParams
	}

	if p.WebhookURL != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("webhook_url", p.WebhookURL))
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
func (p *WebhookEndpointListParams) Encode() (string, error) {
	return p.toListParams().Encode()
}

// WebhookEndpointIter paginates through an installation's webhook endpoints. It
// behaves like Iter but exposes the current element as a typed
// *WebhookEndpoint.
type WebhookEndpointIter struct {
	*Iter
}

// WebhookEndpoint returns the webhook endpoint the iterator currently points to.
func (it *WebhookEndpointIter) WebhookEndpoint() *WebhookEndpoint {
	endpoint, _ := it.Current().(*WebhookEndpoint)
	return endpoint
}

func webhookEndpointsPath(installationID string) string {
	return fmt.Sprintf("/partner/installations/%s/webhook_endpoints", installationID)
}

func (c installationClient) ListWebhookEndpoints(installationID string, params *WebhookEndpointListParams) *WebhookEndpointIter {
	iter := GetIter(params.toListParams(), func(lp *ListParams) ([]interface{}, ListMeta, error) {
		var meta ListMeta

		if installationID == "" {
			return nil, meta, errors.New("installation_id required")
		}

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		path := webhookEndpointsPath(installationID)
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

func (c installationClient) CreateWebhookEndpoint(installationID string, params *WebhookEndpointParams) (*WebhookEndpoint, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	endpoint := &WebhookEndpoint{}
	if _, err := c.B.Call("POST", webhookEndpointsPath(installationID), c.resolveKey(), params, endpoint,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (c installationClient) UpdateWebhookEndpoint(installationID, webhookEndpointID string, params *WebhookEndpointParams) (*WebhookEndpoint, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}
	if webhookEndpointID == "" {
		return nil, errors.New("webhook_endpoint_id required")
	}

	endpoint := &WebhookEndpoint{}
	if _, err := c.B.Call("PATCH", fmt.Sprintf("%s/%s", webhookEndpointsPath(installationID), webhookEndpointID), c.resolveKey(), params, endpoint,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (c installationClient) DeleteWebhookEndpoint(installationID, webhookEndpointID string) error {
	if installationID == "" {
		return errors.New("installation_id required")
	}
	if webhookEndpointID == "" {
		return errors.New("webhook_endpoint_id required")
	}

	if _, err := c.B.Call("DELETE", fmt.Sprintf("%s/%s", webhookEndpointsPath(installationID), webhookEndpointID), c.resolveKey(), nil, nil,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return err
	}
	return nil
}
