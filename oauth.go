package hint

import (
	"errors"
	"time"
)

type OAuthParams struct {
	GrantType string `json:"grant_type"`
	Code      string `json:"code"`
}

func (p *OAuthParams) Validate() error {
	if p.Code == "" {
		return errors.New("code required")
	} else if p.GrantType == "" {
		return errors.New("grant_type required")
	}
	return nil
}

type PracticeGrant struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	TokenType    string     `json:"token_type"`
	RefreshToken *string    `json:"refresh_token"`
	ExpiresIn    *time.Time `json:"expires_in"`
	Practice     *Practice  `json:"practice"`
	AccessToken  string     `json:"access_token"`
}

type OAuthClient interface {
	GrantAPIKey(code string) (*PracticeGrant, error)
}

type oauthClient struct {
	B   Backend
	Key string
}

// resolveKey returns the partner API key used to authenticate calls, preferring
// the client's own key and otherwise falling back to the package-global Key at
// call time. The call-time fallback preserves the behavior of the default
// client, which is constructed at package init before callers assign Key.
func (c oauthClient) resolveKey() string {
	if c.Key != "" {
		return c.Key
	}
	return Key
}

func (c oauthClient) GrantAPIKey(code string) (*PracticeGrant, error) {

	var grant PracticeGrant
	if _, err := c.B.Call("POST", "/oauth/tokens", c.resolveKey(), &OAuthParams{
		GrantType: "authorization_code",
		Code:      code,
	}, &grant); err != nil {
		return nil, err
	}

	return &grant, nil
}
