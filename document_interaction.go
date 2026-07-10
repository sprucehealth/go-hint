package hint

import (
	"errors"
	"fmt"
	"time"
)

// Allowed values for DocumentInteractionParams.Status / DocumentInteraction.Status.
const (
	DocumentInteractionStatusDraft    = "draft"
	DocumentInteractionStatusSigned   = "signed"
	DocumentInteractionStatusAddended = "addended"
	DocumentInteractionStatusDeleted  = "deleted"
	DocumentInteractionStatusFailed   = "failed"
)

// DocumentInteractionParams represents the fields used to create a document interaction on a patient.
type DocumentInteractionParams struct {
	Status                  string     `json:"status"`
	Title                   string     `json:"title,omitempty"`
	Body                    string     `json:"body,omitempty"`
	Attachments             []string   `json:"attachments,omitempty"`
	EventTimestamp          *time.Time `json:"event_timestamp,omitempty"`
	IntegrationRecordID     string     `json:"integration_record_id,omitempty"`
	IntegrationErrorMessage string     `json:"integration_error_message,omitempty"`
	IntegrationWebLink      string     `json:"integration_web_link,omitempty"`
}

// Validate ensures that the required fields for creating a document interaction are present.
func (p *DocumentInteractionParams) Validate() error {
	if p.Status == "" {
		return errors.New("status required")
	}
	return nil
}

// DocumentInteraction represents a document interaction recorded against a patient.
type DocumentInteraction struct {
	ID                      string     `json:"id"`
	Body                    string     `json:"body,omitempty"`
	EventTimestamp          time.Time  `json:"event_timestamp,omitempty"`
	PatientAccess           bool       `json:"patient_access,omitempty"`
	PatientID               string     `json:"patient_id,omitempty"`
	Status                  string     `json:"status,omitempty"`
	Title                   string     `json:"title,omitempty"`
	Type                    string     `json:"type,omitempty"`
	IntegrationErrorMessage string     `json:"integration_error_message,omitempty"`
	IntegrationLastSyncedAt *time.Time `json:"integration_last_synced_at,omitempty"`
	IntegrationRecordID     string     `json:"integration_record_id,omitempty"`
	IntegrationSyncStatus   string     `json:"integration_sync_status,omitempty"`
	IntegrationWebLink      string     `json:"integration_web_link,omitempty"`
}

// DocumentInteractionClient represents the interface for creating document interactions on a patient.
type DocumentInteractionClient interface {
	Create(practiceKey, patientID string, params *DocumentInteractionParams) (*DocumentInteraction, error)
}

type documentInteractionClient struct {
	B   Backend
	Key string
}

// NewDocumentInteractionClient returns an implementation of DocumentInteractionClient.
func NewDocumentInteractionClient(backend Backend, key string) DocumentInteractionClient {
	return &documentInteractionClient{
		B:   backend,
		Key: key,
	}
}

func (c documentInteractionClient) Create(practiceKey, patientID string, params *DocumentInteractionParams) (*DocumentInteraction, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}
	if patientID == "" {
		return nil, errors.New("patient_id required")
	}

	documentInteraction := &DocumentInteraction{}
	if _, err := c.B.Call("POST", fmt.Sprintf("/provider/patients/%s/interactions/document", patientID), practiceKey, params, documentInteraction); err != nil {
		return nil, err
	}
	return documentInteraction, nil
}
