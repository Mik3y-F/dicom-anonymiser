package healthcare

import (
	"context"
	"fmt"

	"google.golang.org/api/healthcare/v1"
)

// constants and defaults
const (
	projectID = "GCLOUD_PROJECT_ID"
	location  = "GCLOUD_PROJECT_LOCATION"
	datasetID = "GCLOUD_PROJECT_DATASET_ID"
)

// DicomAPI represents a healthcare implementation of dicom.DicomService
type DicomAPI struct {
	HealthcareService *healthcare.Service
	StoreService      *healthcare.ProjectsLocationsDatasetsDicomStoresService
}

// NewDicomAPI returns a new instance of DicomAPI
func NewDicomAPI(ctx context.Context) (*DicomAPI, error) {

	healthcareService, err := healthcare.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("healthcare.NewService: %v", err)
	}

	dicomStoreService := healthcareService.Projects.Locations.Datasets.DicomStores

	dicomAPI := &DicomAPI{
		HealthcareService: healthcareService,
		StoreService:      dicomStoreService,
	}
	return dicomAPI, nil
}
