package healthcare

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"ioutil"

	"gitlab.com/medical-research/dicom-deidentifier/dicom"
	healthcare "google.golang.org/api/healthcare/v1"
)

// constants and defaults
const (
	projectID               = "GCLOUD_PROJECT_ID"
	location                = "GCLOUD_PROJECT_LOCATION"
	datasetID               = "GCLOUD_PROJECT_DATASET_ID"
	sourceDicomStoreID      = "GCLOUD_SOURCE_DICOM_STORE"
	destinationDicomStoreID = "GCLOUD_DESTINATION_DICOM_STORE"
	dicomWebPath            = "GCLOUD_DICOM_WEB_PATH" // TODO: investigate need for this
)

// Service represents a healthcare implementation of dicom.Service
type Service struct {
	LogOutput io.Writer
}

// StoreDICOMImageInstances uploads DICOM instances to the gcloud healthcare api
func (s *Service) StoreDICOMImageInstances(df []*dicom.ImageFile) error {
	for _, image := range df {
		err := s.StoreDICOMImageInstance(image.Path)
		if err != nil {
			// TODO: Think about a rollback strategy for when there's an error when creating DICOM instances
			return fmt.Errorf("failed to complete storing dicom image instances: %s", err)
		}
	}

	return nil
}

// StoreDICOMImageInstance stores the given dicomFile with the dicomWebPath.
func (s *Service) StoreDICOMImageInstance(dicomFile string) error {
	ctx := context.Background()

	dicomData, err := ioutil.ReadFile(dicomFile)
	if err != nil {
		return fmt.Errorf("ReadFile: %v", err)
	}

	healthcareService, err := healthcare.NewService(ctx)
	if err != nil {
		return fmt.Errorf("healthcare.NewService: %v", err)
	}

	storesService := healthcareService.Projects.Locations.Datasets.DicomStores

	parent := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/dicomStores/%s", projectID, location, datasetID, sourceDicomStoreID)

	call := storesService.StoreInstances(parent, dicomWebPath, bytes.NewReader(dicomData))
	call.Header().Set("Content-Type", "application/dicom")
	resp, err := call.Do()
	if err != nil {
		return fmt.Errorf("StoreInstances: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response: %v", err)
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("StoreInstances: status %d %s: %s", resp.StatusCode, resp.Status, respBytes)
	}
	fmt.Fprintf(s.LogOutput, "%s", respBytes)
	return nil
}
