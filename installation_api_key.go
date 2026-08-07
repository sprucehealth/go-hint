package hint

import (
	"errors"
	"fmt"
)

// APIKeyParams is the request body for creating or updating an installation's
// API key.
type APIKeyParams struct {
	// Label is the descriptive name for the key.
	Label string `json:"label,omitempty"`
}

// Validate ensures the required fields for writing an API key are present. The
// label is optional on update as far as the API is concerned, but an update
// without it is a no-op, so it is required here for both operations.
func (p *APIKeyParams) Validate() error {
	if p.Label == "" {
		return errors.New("label required")
	}
	return nil
}

// APIKeyListParams are the filters accepted by
// GET /partner/installations/{installation_id}/api_keys. A nil *APIKeyListParams
// applies no filters and uses the server's default page size.
type APIKeyListParams struct {
	// Label narrows the results to keys with exactly this label.
	Label string
	// CreatedAt filters on the key's creation timestamp. Operands must be ISO8601
	// timestamps, and the operators are limited to gt, gte, lt and lte.
	CreatedAt []*Operation
	// LastUsedAt filters on the key's last use timestamp, in the same form as
	// CreatedAt.
	LastUsedAt []*Operation
	// DeactivatedAt filters on the key's deactivation timestamp, in the same form
	// as CreatedAt. It also accepts IsPresent, which selects only the deactivated
	// keys (true) or only the live ones (false).
	DeactivatedAt []*Operation
	// Offset is the starting position for pagination. The iterator returned by
	// ListAPIKeys advances it automatically when fetching subsequent pages.
	Offset uint64
	// Limit constrains the number of keys returned per page. Zero omits the
	// parameter so the server default applies.
	Limit uint64
}

// toListParams converts the typed filters into the generic ListParams the
// backend query encoder and the Iter pagination both operate on. A nil receiver
// yields empty (unfiltered) params.
func (p *APIKeyListParams) toListParams() *ListParams {
	listParams := &ListParams{}
	if p == nil {
		return listParams
	}

	if p.Label != "" {
		listParams.Items = append(listParams.Items, equalToQueryItem("label", p.Label))
	}
	if len(p.CreatedAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "created_at",
			Operations: p.CreatedAt,
		})
	}
	if len(p.LastUsedAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "last_used_at",
			Operations: p.LastUsedAt,
		})
	}
	if len(p.DeactivatedAt) > 0 {
		listParams.Items = append(listParams.Items, &QueryItem{
			Field:      "deactivated_at",
			Operations: p.DeactivatedAt,
		})
	}
	listParams.Offset = p.Offset
	listParams.Limit = p.Limit

	return listParams
}

// Encode returns the query string for the params, in the format documented at
// https://developers.hint.com/reference/making-requests#advanced-querying.
func (p *APIKeyListParams) Encode() (string, error) {
	return p.toListParams().Encode()
}

// APIKeyIter paginates through an installation's API keys. It behaves like Iter
// but exposes the current element as a typed *APIKey.
//
// The list endpoint does not return the keys' IDs or tokens, so the elements
// carry only the key metadata (Label, TokenLast4 and the timestamps).
type APIKeyIter struct {
	*Iter
}

// APIKey returns the API key the iterator currently points to.
func (it *APIKeyIter) APIKey() *APIKey {
	apiKey, _ := it.Current().(*APIKey)
	return apiKey
}

func apiKeysPath(installationID string) string {
	return fmt.Sprintf("/partner/installations/%s/api_keys", installationID)
}

func (c installationClient) ListAPIKeys(installationID string, params *APIKeyListParams) *APIKeyIter {
	iter := GetIter(params.toListParams(), func(lp *ListParams) ([]interface{}, ListMeta, error) {
		var meta ListMeta

		if installationID == "" {
			return nil, meta, errors.New("installation_id required")
		}

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		path := apiKeysPath(installationID)
		if encodedParams != "" {
			path += "?" + encodedParams
		}

		var apiKeys []*APIKey
		resHeaders, err := c.B.Call("GET", path, c.resolveKey(), nil, &apiKeys,
			WithHeader(hintVersionHeader, HintVersionMarketplace))
		if err != nil {
			return nil, meta, err
		}

		if meta, err = parseListMeta(resHeaders); err != nil {
			return nil, meta, err
		}

		ret := make([]interface{}, len(apiKeys))
		for i, apiKey := range apiKeys {
			ret[i] = apiKey
		}

		return ret, meta, nil
	})

	return &APIKeyIter{Iter: iter}
}

func (c installationClient) CreateAPIKey(installationID string, params *APIKeyParams) (*APIKey, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}

	apiKey := &APIKey{}
	if _, err := c.B.Call("POST", apiKeysPath(installationID), c.resolveKey(), params, apiKey,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (c installationClient) UpdateAPIKey(installationID, apiKeyID string, params *APIKeyParams) (*APIKey, error) {
	if installationID == "" {
		return nil, errors.New("installation_id required")
	}
	if apiKeyID == "" {
		return nil, errors.New("api_key_id required")
	}

	apiKey := &APIKey{}
	if _, err := c.B.Call("PATCH", fmt.Sprintf("%s/%s", apiKeysPath(installationID), apiKeyID), c.resolveKey(), params, apiKey,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (c installationClient) DeleteAPIKey(installationID, apiKeyID string) error {
	if installationID == "" {
		return errors.New("installation_id required")
	}
	if apiKeyID == "" {
		return errors.New("api_key_id required")
	}

	if _, err := c.B.Call("DELETE", fmt.Sprintf("%s/%s", apiKeysPath(installationID), apiKeyID), c.resolveKey(), nil, nil,
		WithHeader(hintVersionHeader, HintVersionMarketplace)); err != nil {
		return err
	}
	return nil
}
