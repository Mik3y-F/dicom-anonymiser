package healthcare

import (
	"context"
	"fmt"
	"time"

	dicomdeidentifier "gitlab.com/medical-research/dicom-deidentifier"
	"google.golang.org/api/healthcare/v1"
)

// Ensure service implements interface.
var _ dicomdeidentifier.DicomStoreService = (*DicomStoreService)(nil)

// DicomStoreService represents a service for managing DicomStores
type DicomStoreService struct {
	DicomAPI *DicomAPI
}

// NewDicomStoreService returns a new instance of DicomStoreService
func NewDicomStoreService(dicomAPI *DicomAPI) *DicomStoreService {
	return &DicomStoreService{
		DicomAPI: dicomAPI,
	}
}

// CreateDicomStore creates special storage abstractions in the cloud known as dicom stores
// The dicom stores will hold the various dicom instances created
func (s *DicomStoreService) CreateDicomStore(ctx context.Context, dicomStoreID string) (*dicomdeidentifier.DicomStore, error) {

	store := &healthcare.DicomStore{}
	parent := fmt.Sprintf("projects/%s/locations/%s/datasets/%s", projectID, location, datasetID)

	resp, err := s.DicomAPI.StoreService.Create(parent, store).DicomStoreId(dicomStoreID).Do()
	if err != nil {
		return nil, fmt.Errorf("Create: %v", err)
	}

	fmt.Printf("Created DICOM store: %q\n", resp.Name)
	return nil, nil
}

// DeleteDicomStore Deletes an existing dicom store
func (s *DicomStoreService) DeleteDicomStore(ctx context.Context, dicomStoreID string) error {

	name := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/dicomStores/%s", projectID, location, datasetID, dicomStoreID)
	if _, err := s.DicomAPI.StoreService.Delete(name).Do(); err != nil {
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
func (s *DicomStoreService) GetDicomStoreList(ctx context.Context) ([]*dicomdeidentifier.DicomStore, error) {

	parent := fmt.Sprintf("projects/%s/locations/%s/datasets/%s", projectID, location, datasetID)

	resp, err := s.DicomAPI.StoreService.List(parent).Do()
	if err != nil {
		return nil, fmt.Errorf("List: %v", err)
	}

	dicomStores := []*dicomdeidentifier.DicomStore{}

	for _, s := range resp.DicomStores {
		d := dicomdeidentifier.DicomStore{
			StoreID: s.Name,
		}
		dicomStores = append(dicomStores, &d)
	}
	return dicomStores, fmt.Errorf("dicom store list could not be retreived")
}

// DeidentifyDicomStore Strips the P.I.I(Personally Identifiable Information) embedded in the dicom instances
// Deidentified dicom instances will be stored in the destinationDicomStoreProvided
func (s *DicomStoreService) DeidentifyDicomStore(ctx context.Context, sourceDicomStore, destinationDicomStore *dicomdeidentifier.DicomStore) error {

	datasetsService := s.DicomAPI.HealthcareService.Projects.Locations.Datasets

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)

	req := &healthcare.DeidentifyDatasetRequest{
		DestinationDataset: fmt.Sprintf("%s/datasets/%s", parent, destinationDicomStore.StoreID),
		Config: &healthcare.DeidentifyConfig{
			Dicom: &healthcare.DicomConfig{
				KeepList: &healthcare.TagFilterList{
					Tags: []string{},
				},
				FilterProfile: "MINIMAL_KEEP_LIST_PROFILE",
			},
			Image: &healthcare.ImageConfig{
				TextRedactionMode: "REDUCT_SENSITIVE_TEXT",
			},
		},
	}

	sourceName := fmt.Sprintf("%s/datasets/%s", parent, sourceDicomStore.StoreID)
	resp, err := datasetsService.Deidentify(sourceName, req).Do()
	if err != nil {
		return fmt.Errorf("Deidentify: %v", err)
	}

	// Wait for the deidentification operation to finish.
	operationService := s.DicomAPI.HealthcareService.Projects.Locations.Datasets.Operations
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
