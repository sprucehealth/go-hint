package hint

type Partner struct {
	WebhookURL             string `json:"webhook_url"`
	MarketingURL           string `json:"marketing_url"`
	SupportURL             string `json:"support_url"`
	RedirectURL            string `json:"redirect_url"`
	ProductDescription     string `json:"product_description"`
	IntegrationDescription string `json:"integration_description"`
	Email                  string `json:"email"`
	LogoURL                string `json:"logo_url"`
	Name                   string `json:"name"`
}

type PartnerParams struct {
	Partner *Partner `json:"partner"`
}

func (p *PartnerParams) Validate() error {
	return nil
}

type PartnerClient interface {
	// Get returns the partner object
	Get() (*Partner, error)
	// Update updates the partner object based on the params
	Update(params *PartnerParams) (*Partner, error)
}

type partnerClient struct {
	B   Backend
	Key string
}

func NewPartnerClient(backend Backend, key string) PartnerClient {
	return &partnerClient{
		B:   backend,
		Key: key,
	}
}

// resolveKey mirrors oauthClient.resolveKey: prefer the per-client key,
// otherwise fall back to the package-global Key at call time.
func (c partnerClient) resolveKey() string {
	if c.Key != "" {
		return c.Key
	}
	return Key
}

func (c partnerClient) Get() (*Partner, error) {
	var partner Partner
	if _, err := c.B.Call("GET", "/partner", c.resolveKey(), nil, &partner); err != nil {
		return nil, err
	}

	return &partner, nil
}

func (c partnerClient) Update(params *PartnerParams) (*Partner, error) {
	var partner Partner
	if _, err := c.B.Call("PATCH", "/partner", c.resolveKey(), params, &partner); err != nil {
		return nil, err
	}

	return &partner, nil
}
