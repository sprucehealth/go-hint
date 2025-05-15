package hint

import (
	"errors"
	"fmt"
)

// IntegrationRecord represents a patient registered as part of a practice on hint.
type IntegrationRecord struct {
	IntegrationRecordID string              `json:"integration_record_id"`
	IntegrationPartner  *IntegrationPartner `json:"partner"`
}

type IntegrationPartner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IntegrationRecordsClient interface {
	Get(practiceKey, id string) ([]*IntegrationRecord, error)
}

type integrationRecordsClient struct {
	B   Backend
	Key string
}

func NewIntegrationRecordsClient(backend Backend, key string) IntegrationRecordsClient {
	return &integrationRecordsClient{
		B:   backend,
		Key: key,
	}
}

func (c integrationRecordsClient) Get(practiceKey, patientID string) ([]*IntegrationRecord, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}

	integrationRecords := []*IntegrationRecord{}
	if _, err := c.B.Call("GET", fmt.Sprintf("/provider/patients/%s/integration_records", patientID), practiceKey, nil, &integrationRecords); err != nil {
		return nil, err
	}
	return integrationRecords, nil
}
