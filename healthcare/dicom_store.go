package healthcare

import (
	"context"
	"fmt"
	"time"

	dcmd "gitlab.com/medical-research/dicom-deidentifier"
	"google.golang.org/api/healthcare/v1"
)

// Ensure service implements interface.
var _ dcmd.DicomStoreService = (*DicomStoreService)(nil)

// DicomStoreService represents a service for managing DicomStores
type DicomStoreService struct {
	GoogleDicomAPI *GoogleDicomAPI
}

// NewDicomStoreService returns a new instance of DicomStoreService
func NewDicomStoreService(dicomAPI *GoogleDicomAPI) *DicomStoreService {
	return &DicomStoreService{
		GoogleDicomAPI: dicomAPI,
	}
}

// CreateDicomStore creates special storage abstractions in the cloud known as dicom stores
// The dicom stores will hold the various dicom instances created
func (s *DicomStoreService) CreateDicomStore(ctx context.Context, dicomStoreID string) (*dcmd.DicomStore, error) {

	store := &healthcare.DicomStore{}
	parent := s.GoogleDicomAPI.Dataset.Name

	resp, err := s.GoogleDicomAPI.StoreService.Create(parent, store).DicomStoreId(dicomStoreID).Do()
	if err != nil {
		return nil, fmt.Errorf("Create: %v", err)
	}

	fmt.Printf("Created DICOM store: %q\n", resp.Name)
	return nil, nil
}

// DeleteDicomStore Deletes an existing dicom store
func (s *DicomStoreService) DeleteDicomStore(ctx context.Context, dicomStoreID string) error {

	name := fmt.Sprintf("%s/dicomStores/%s", s.GoogleDicomAPI.Dataset.Name, dicomStoreID)
	if _, err := s.GoogleDicomAPI.StoreService.Delete(name).Do(); err != nil {
		return fmt.Errorf("Delete: %v", err)
	}

	fmt.Printf("Deleted DICOM store: %q\n", name)
	return nil
}

// GenerateDicomStoreID generates a unique Dicom store name
func (s *DicomStoreService) GenerateDicomStoreID(ctx context.Context) (string, error) {
	return "", fmt.Errorf("unable to generate dicom store name")
}

// GetDicomStoreList retreives a list of all dicom stores created
func (s *DicomStoreService) GetDicomStoreList(ctx context.Context) ([]*dcmd.DicomStore, error) {

	parent := s.GoogleDicomAPI.Dataset.Name

	resp, err := s.GoogleDicomAPI.StoreService.List(parent).Do()
	if err != nil {
		return nil, fmt.Errorf("List: %v", err)
	}

	dicomStores := []*dcmd.DicomStore{}

	for _, s := range resp.DicomStores {
		d := dcmd.DicomStore{
			StoreID: s.Name,
		}
		dicomStores = append(dicomStores, &d)
	}
	return dicomStores, fmt.Errorf("dicom store list could not be retreived")
}

// DeidentifyDicomStore Strips the P.I.I(Personally Identifiable Information) embedded in the dicom instances
// Deidentified dicom instances will be stored in the destinationDicomStoreProvided
func (s *DicomStoreService) DeidentifyDicomStore(ctx context.Context, sourceDicomStore, destinationDicomStore *dcmd.DicomStore) error {

	datasetsService := s.GoogleDicomAPI.HealthcareService.Projects.Locations.Datasets.DicomStores

	req := &healthcare.DeidentifyDicomStoreRequest{
		DestinationStore: fmt.Sprintf("%s/dicomStores/%s", s.GoogleDicomAPI.Dataset.Name, destinationDicomStore.StoreID),
		Config: &healthcare.DeidentifyConfig{
			Dicom: &healthcare.DicomConfig{
				FilterProfile: "MINIMAL_KEEP_LIST_PROFILE",
			},
			Image: &healthcare.ImageConfig{
				TextRedactionMode: "REDACT_SENSITIVE_TEXT",
			},
		},
	}

	sourceName := fmt.Sprintf("%s/dicomStores/%s", s.GoogleDicomAPI.Dataset.Name, sourceDicomStore.StoreID)
	resp, err := datasetsService.Deidentify(sourceName, req).Do()
	if err != nil {
		return fmt.Errorf("Deidentify: %v", err)
	}

	// Wait for the deidentification operation to finish.
	operationService := s.GoogleDicomAPI.HealthcareService.Projects.Locations.Datasets.Operations
	for {
		op, err := operationService.Get(resp.Name).Do()
		if err != nil {
			return fmt.Errorf("operationService.Get: %v", err)
		}
		if !op.Done {
			time.Sleep(1 * time.Second)
			continue
		}
		if op.Error != nil {
			return fmt.Errorf("deidentify operation error: %v", *op.Error)
		}
		fmt.Printf("Created de-identified dataset %s from %s\n", resp.Name, sourceName)
		return nil
	}

}
