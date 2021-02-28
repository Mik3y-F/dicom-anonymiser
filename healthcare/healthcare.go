package healthcare

import (
	"context"
	"fmt"

	dcmd "gitlab.com/medical-research/dicom-deidentifier"
	"google.golang.org/api/healthcare/v1"
)

// constants and defaults
const (
	ProjectID = "GCP_PROJECT"
	Location  = "GCLOUD_PROJECT_LOCATION"
	DatasetID = "GCLOUD_PROJECT_DATASET_ID"
)

// GoogleDicomAPI represents a healthcare implementation of dicom.DicomService
type GoogleDicomAPI struct {
	HealthcareService *healthcare.Service
	StoreService      *healthcare.ProjectsLocationsDatasetsDicomStoresService
	Dataset           *healthcare.Dataset
}

// NewDicomAPI returns a new instance of DicomAPI
func NewDicomAPI(ctx context.Context) (*GoogleDicomAPI, error) {

	p := dcmd.MustGetEnvVar(ProjectID)
	l := dcmd.MustGetEnvVar(Location)
	d := dcmd.MustGetEnvVar(DatasetID)

	datasetName := fmt.Sprintf("projects/%s/locations/%s/datasets/%s", p, l, d)

	healthcareService, err := healthcare.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("healthcare.NewService: %v", err)
	}

	dicomStoreService := healthcareService.Projects.Locations.Datasets.DicomStores

	dicomAPI := &GoogleDicomAPI{
		HealthcareService: healthcareService,
		StoreService:      dicomStoreService,

		Dataset: &healthcare.Dataset{
			Name: datasetName,
		},
	}
	return dicomAPI, nil
}
