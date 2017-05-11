package hint

type client struct {
	Patient      PatientClient
	OAuth        OAuthClient
	Partner      PartnerClient
	Practitioner PractitionerClient
}

var defaultClient = getC()

func getC() *client {
	return &client{
		Patient:      &patientClient{B: GetBackend(), Key: Key},
		OAuth:        &oauthClient{B: GetBackend(), Key: Key},
		Partner:      &partnerClient{B: GetBackend(), Key: Key},
		Practitioner: &practitionerClient{B: GetBackend(), Key: Key},
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
}

// NewPracticeClient returns an implementation of practiceClient
func NewPracticeClient(accessToken string) PracticeClient {
	return &practiceClient{
		accessToken: accessToken,
		client:      getC(),
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
