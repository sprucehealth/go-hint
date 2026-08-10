package hint

type client struct {
	Patient                 PatientClient
	OAuth                   OAuthClient
	Partner                 PartnerClient
	Practitioner            PractitionerClient
	IntegrationRecords      IntegrationRecordsClient
	DocumentInteractions    DocumentInteractionClient
	Installations           InstallationClient
	PartnerBackends         PartnerBackendClient
	PartnerWebhookEndpoints PartnerWebhookEndpointClient
}

var defaultClient = getC()

// clientOptions holds the configurable settings applied when constructing a
// client. Zero values indicate that the corresponding default should be used.
type clientOptions struct {
	baseURL    string
	partnerKey string
}

// Option configures optional behavior of a client created via NewPracticeClient
// or NewOAuthClient.
type Option func(*clientOptions)

// WithBaseURL configures the base API URL used for all calls made by the client.
// When this option is not provided, the base URL defaults to the value derived
// from the Testing flag (staging when true, production otherwise).
//
// The provided URL is used as-is as the prefix for request paths, so it should
// include any scheme, host, and base path (e.g. "https://api.hint.com/api").
func WithBaseURL(baseURL string) Option {
	return func(o *clientOptions) {
		o.baseURL = baseURL
	}
}

// WithPartnerKey configures the Hint partner API key used by the client (e.g. a
// sandbox partner key). When this option is not provided, the client falls back
// to the package-global Key at call time.
func WithPartnerKey(key string) Option {
	return func(o *clientOptions) {
		o.partnerKey = key
	}
}

func getC(opts ...Option) *client {
	var options clientOptions
	for _, opt := range opts {
		opt(&options)
	}

	backend := getBackend(options.baseURL)
	return &client{
		Patient:                 &patientClient{B: backend, Key: Key},
		OAuth:                   &oauthClient{B: backend, Key: options.partnerKey},
		Partner:                 &partnerClient{B: backend, Key: options.partnerKey},
		Practitioner:            &practitionerClient{B: backend, Key: Key},
		IntegrationRecords:      &integrationRecordsClient{B: backend, Key: Key},
		DocumentInteractions:    &documentInteractionClient{B: backend, Key: Key},
		Installations:           &installationClient{B: backend, Key: options.partnerKey},
		PartnerBackends:         &partnerBackendClient{B: backend, Key: options.partnerKey},
		PartnerWebhookEndpoints: &partnerWebhookEndpointClient{B: backend, Key: options.partnerKey},
	}
}

type practiceClient struct {
	accessToken string
	client      *client
}

// PracticeClient represents the practice specific top level operations exposed by the hint api
type PracticeClient interface {
	NewPatient(params *PatientParams) (*Patient, error)
	GetPatient(id string) (*Patient, error)
	UpdatePatient(id string, params *PatientParams) (*Patient, error)
	DeletePatient(id string) error
	ListPatient(params *ListParams) *Iter
	ListAllPractitioners() ([]*Practitioner, error)
	GetIntegrationRecords(patientID string) ([]*IntegrationRecord, error)
	CreateDocumentInteraction(patientID string, params *DocumentInteractionParams) (*DocumentInteraction, error)
}

// NewPracticeClient returns an implementation of practiceClient. Options may be
// supplied to customize the client, for example WithBaseURL to override the base
// API URL used for calls. When no options are provided the client uses the base
// URL derived from the Testing flag.
func NewPracticeClient(accessToken string, opts ...Option) PracticeClient {
	return &practiceClient{
		accessToken: accessToken,
		client:      getC(opts...),
	}
}

// SetPatientClient enables caller to provide a particular implementation of the patient client for mocking purposes.
func SetPatientClient(c PatientClient) {
	defaultClient.Patient = c
}

// SetOAuthClient enables caller to provide a particular implementation of the oauth client for mocking purposes.
func SetOAuthClient(c OAuthClient) {
	defaultClient.OAuth = c
}

// SetPartnerClient enables caller to provide a particular implementation of the partner client for mocking purposes.
func SetPartnerClient(c PartnerClient) {
	defaultClient.Partner = c
}

// SetPractitionerClient enables caller to provide a particular implementation of the practitioner client for mocking purposes.
func SetPractitionerClient(c PractitionerClient) {
	defaultClient.Practitioner = c
}

// SetDocumentInteractionsClient enables caller to provide a particular implementation of the document interactions client for mocking purposes.
func SetDocumentInteractionsClient(c DocumentInteractionClient) {
	defaultClient.DocumentInteractions = c
}

// SetInstallationsClient enables caller to provide a particular implementation of the installations client for mocking purposes.
func SetInstallationsClient(c InstallationClient) {
	defaultClient.Installations = c
}

// SetPartnerBackendsClient enables caller to provide a particular implementation of the partner backends client for mocking purposes.
func SetPartnerBackendsClient(c PartnerBackendClient) {
	defaultClient.PartnerBackends = c
}

// SetPartnerWebhookEndpointsClient enables caller to provide a particular implementation of the partner webhook endpoints client for mocking purposes.
func SetPartnerWebhookEndpointsClient(c PartnerWebhookEndpointClient) {
	defaultClient.PartnerWebhookEndpoints = c
}

// NewPatient creates a new patient based on the params.
func NewPatient(practiceKey string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.New(practiceKey, params)
}

// NewPatient creates a new patient based on the params.
func (c *practiceClient) NewPatient(params *PatientParams) (*Patient, error) {
	return c.client.Patient.New(c.accessToken, params)
}

// GetPatient gets an existing patient in the practice account.
func GetPatient(practiceKey, id string) (*Patient, error) {
	return defaultClient.Patient.Get(practiceKey, id)
}

// GetPatient gets an existing patient in the practice account.
func (c *practiceClient) GetPatient(id string) (*Patient, error) {
	return c.client.Patient.Get(c.accessToken, id)
}

// UpdatePatient updates an existing patient based on the params.
func UpdatePatient(practiceKey, id string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.Update(practiceKey, id, params)
}

// UpdatePatient updates an existing patient based on the params.
func (c *practiceClient) UpdatePatient(id string, params *PatientParams) (*Patient, error) {
	return c.client.Patient.Update(c.accessToken, id, params)
}

// DeletePatient deletes a patient based on the id.
func DeletePatient(practiceKey, id string) error {
	return defaultClient.Patient.Delete(practiceKey, id)
}

// DeletePatient deletes a patient based on the id.
func (c *practiceClient) DeletePatient(id string) error {
	return c.client.Patient.Delete(c.accessToken, id)
}

// ListPatient returns an iterator that can be used to paginate through the list of patients
// based on the iterator.
func ListPatient(practiceKey string, params *ListParams) *Iter {
	return defaultClient.Patient.List(practiceKey, params)
}

// ListPatient returns an iterator that can be used to paginate through the list of patients
// based on the iterator.
func (c *practiceClient) ListPatient(params *ListParams) *Iter {
	return c.client.Patient.List(c.accessToken, params)
}

// NewOAuthClient returns an implementation of OAuthClient. Options may be
// supplied to customize the client, for example WithBaseURL to override the
// base API URL used for the token exchange (e.g. SandboxAPIURL) and
// WithPartnerKey to override the partner API key used for authentication.
// When no options are provided the client uses the base URL derived from the
// Testing flag and the package-global Key.
func NewOAuthClient(opts ...Option) OAuthClient {
	return getC(opts...).OAuth
}

// GrantAPIKey exchanges the OAuth token for a practice API key.
func GrantAPIKey(code string) (*PracticeGrant, error) {
	return defaultClient.OAuth.GrantAPIKey(code)
}

// GetPartner returns information about the partner.
func GetPartner() (*Partner, error) {
	return defaultClient.Partner.Get()
}

// UpdatePartner enables updating partner information and returns the updated partner.
func UpdatePartner(params *PartnerParams) (*Partner, error) {
	return defaultClient.Partner.Update(params)
}

// ListAllPractitioners lists all practitioners part of the practice.
func ListAllPractitioners(practiceKey string) ([]*Practitioner, error) {
	return defaultClient.Practitioner.List(practiceKey)
}

// ListAllPractitioners lists all practitioners part of the practice.
func (c *practiceClient) ListAllPractitioners() ([]*Practitioner, error) {
	return c.client.Practitioner.List(c.accessToken)
}

// GetIntegrationRecords gets all integration records for a patient.
func GetIntegrationRecords(practiceKey, patientID string) ([]*IntegrationRecord, error) {
	return defaultClient.IntegrationRecords.Get(practiceKey, patientID)
}

// GetIntegrationRecords gets all integration records for a patient.
func (c *practiceClient) GetIntegrationRecords(patientID string) ([]*IntegrationRecord, error) {
	return c.client.IntegrationRecords.Get(c.accessToken, patientID)
}

// CreateDocumentInteraction creates a document interaction on a patient.
func CreateDocumentInteraction(practiceKey, patientID string, params *DocumentInteractionParams) (*DocumentInteraction, error) {
	return defaultClient.DocumentInteractions.Create(practiceKey, patientID, params)
}

// CreateDocumentInteraction creates a document interaction on a patient.
func (c *practiceClient) CreateDocumentInteraction(patientID string, params *DocumentInteractionParams) (*DocumentInteraction, error) {
	return c.client.DocumentInteractions.Create(c.accessToken, patientID, params)
}

// ListInstallations returns an iterator that paginates through the partner's
// installations (GET /partner/installations) using the package-global Key. A nil
// params lists every installation.
func ListInstallations(params *InstallationListParams) *InstallationIter {
	return defaultClient.Installations.List(params)
}

// GetInstallation returns a single installation by ID using the package-global
// Key. Deactivated installations are returned too.
func GetInstallation(installationID string) (*Installation, error) {
	return defaultClient.Installations.Get(installationID)
}

// ActivateInstallation activates a pending installation using the
// package-global Key and returns the updated installation object.
func ActivateInstallation(installationID string) (*Installation, error) {
	return defaultClient.Installations.Activate(installationID)
}

// PushCredential pushes (or rotates) the credential for an installation using
// the package-global Key and returns the stored credential record.
func PushCredential(installationID string, params *CredentialParams) (*Credential, error) {
	return defaultClient.Installations.PushCredential(installationID, params)
}

// ConnectInstallation activates a pending installation using the authorization
// code issued when the practice installed the product, authenticating with the
// package-global Key, and returns the installation object with its API
// credentials.
func ConnectInstallation(params *ConnectParams) (*Installation, error) {
	return defaultClient.Installations.Connect(params)
}

// DeactivateInstallation deactivates the installation using the package-global
// Key and returns the updated installation object. The practice's other
// installations are not affected.
func DeactivateInstallation(installationID string) (*Installation, error) {
	return defaultClient.Installations.Deactivate(installationID)
}

// ListWebhookEndpoints returns an iterator that paginates through the URLs Hint
// delivers the installation's webhook events to, using the package-global Key. A
// nil params lists every endpoint.
func ListWebhookEndpoints(installationID string, params *WebhookEndpointListParams) *WebhookEndpointIter {
	return defaultClient.Installations.ListWebhookEndpoints(installationID, params)
}

// CreateWebhookEndpoint registers a URL for the installation's webhook events
// using the package-global Key and returns the created endpoint.
func CreateWebhookEndpoint(installationID string, params *WebhookEndpointParams) (*WebhookEndpoint, error) {
	return defaultClient.Installations.CreateWebhookEndpoint(installationID, params)
}

// UpdateWebhookEndpoint points an existing webhook endpoint at a new URL using
// the package-global Key and returns the updated endpoint.
func UpdateWebhookEndpoint(installationID, webhookEndpointID string, params *WebhookEndpointParams) (*WebhookEndpoint, error) {
	return defaultClient.Installations.UpdateWebhookEndpoint(installationID, webhookEndpointID, params)
}

// DeleteWebhookEndpoint removes a webhook endpoint from the installation using
// the package-global Key, so Hint stops delivering the installation's events to
// it.
func DeleteWebhookEndpoint(installationID, webhookEndpointID string) error {
	return defaultClient.Installations.DeleteWebhookEndpoint(installationID, webhookEndpointID)
}

// ListInstallationAPIKeys returns an iterator that paginates through the API
// keys of the installation's practice connection, using the package-global Key.
// A nil params lists every key. The listed keys carry no ID or token, only their
// metadata.
func ListInstallationAPIKeys(installationID string, params *APIKeyListParams) *APIKeyIter {
	return defaultClient.Installations.ListAPIKeys(installationID, params)
}

// CreateInstallationAPIKey issues a new API key for the installation using the
// package-global Key. The returned APIKey.Token holds the full secret and is the
// only time Hint returns it, so it has to be stored on receipt.
func CreateInstallationAPIKey(installationID string, params *APIKeyParams) (*APIKey, error) {
	return defaultClient.Installations.CreateAPIKey(installationID, params)
}

// UpdateInstallationAPIKey relabels an existing API key using the
// package-global Key and returns the updated key.
func UpdateInstallationAPIKey(installationID, apiKeyID string, params *APIKeyParams) (*APIKey, error) {
	return defaultClient.Installations.UpdateAPIKey(installationID, apiKeyID, params)
}

// DeleteInstallationAPIKey removes an API key from the installation's practice
// connection using the package-global Key, so it can no longer be used to
// authenticate.
func DeleteInstallationAPIKey(installationID, apiKeyID string) error {
	return defaultClient.Installations.DeleteAPIKey(installationID, apiKeyID)
}

// ListPartnerBackends returns an iterator that paginates through the partner's
// backends (GET /partner/backends) using the package-global Key. A nil params
// lists every backend.
func ListPartnerBackends(params *PartnerBackendListParams) *PartnerBackendIter {
	return defaultClient.PartnerBackends.List(params)
}

// GetPartnerBackend returns a single partner backend by ID using the
// package-global Key.
func GetPartnerBackend(backendID string) (*PartnerBackend, error) {
	return defaultClient.PartnerBackends.Get(backendID)
}

// UpdatePartnerBackend updates a partner backend's configuration using the
// package-global Key and returns the updated backend.
func UpdatePartnerBackend(backendID string, params *PartnerBackendUpdateParams) (*PartnerBackend, error) {
	return defaultClient.PartnerBackends.Update(backendID, params)
}

// ListPartnerWebhookEndpoints returns an iterator that paginates through the
// partner-level webhook endpoints (GET /partner/webhook_endpoints) using the
// package-global Key. A nil params lists every endpoint.
func ListPartnerWebhookEndpoints(params *PartnerWebhookEndpointListParams) *WebhookEndpointIter {
	return defaultClient.PartnerWebhookEndpoints.List(params)
}

// CreatePartnerWebhookEndpoint registers a URL to receive events for every
// integration connected to the partner, using the package-global Key, and
// returns the created endpoint.
func CreatePartnerWebhookEndpoint(params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error) {
	return defaultClient.PartnerWebhookEndpoints.Create(params)
}

// UpdatePartnerWebhookEndpoint points an existing partner-level webhook
// endpoint at a new URL using the package-global Key and returns the updated
// endpoint.
func UpdatePartnerWebhookEndpoint(webhookEndpointID string, params *PartnerWebhookEndpointParams) (*WebhookEndpoint, error) {
	return defaultClient.PartnerWebhookEndpoints.Update(webhookEndpointID, params)
}

// DeletePartnerWebhookEndpoint removes a partner-level webhook endpoint using
// the package-global Key, so Hint stops delivering events to it.
func DeletePartnerWebhookEndpoint(webhookEndpointID string) error {
	return defaultClient.PartnerWebhookEndpoints.Delete(webhookEndpointID)
}
