package hint

type client struct {
	Patient              PatientClient
	OAuth                OAuthClient
	Partner              PartnerClient
	Practitioner         PractitionerClient
	IntegrationRecords   IntegrationRecordsClient
	DocumentInteractions DocumentInteractionClient
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
		Patient:              &patientClient{B: backend, Key: Key},
		OAuth:                &oauthClient{B: backend, Key: options.partnerKey},
		Partner:              &partnerClient{B: backend, Key: Key},
		Practitioner:         &practitionerClient{B: backend, Key: Key},
		IntegrationRecords:   &integrationRecordsClient{B: backend, Key: Key},
		DocumentInteractions: &documentInteractionClient{B: backend, Key: Key},
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
