package healthcare

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	dicomdeidentifier "gitlab.com/medical-research/dicom-deidentifier"
)

// Ensure service implements interface.
var _ dicomdeidentifier.DicomService = (*DicomService)(nil)

// DicomService represents a service for managing Dicoms
type DicomService struct {
	dicomAPI *DicomAPI
}

// NewDicomService returns a new instance of DicomService
func NewDicomService(dicomAPI *DicomAPI) *DicomService {
	return &DicomService{
		dicomAPI: dicomAPI,
	}
}

// CreateDicomInstances creates dicom instances in the cloud within special abstractions called dicomStores
func (s *DicomService) CreateDicomInstances(ctx context.Context, dicomStore dicomdeidentifier.DicomStore, dicoms ...dicomdeidentifier.Dicom) error {

	for _, dicom := range dicoms {
		dicomData, err := ioutil.ReadFile(dicom.Path)
		if err != nil {
			return fmt.Errorf("ReadFile: %v", err)
		}

		parent := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/dicomStores/%s", projectID, location, datasetID, dicomStore.StoreID)
		dicomWebPath := "studies"

		call := s.dicomAPI.StoreService.StoreInstances(parent, dicomWebPath, bytes.NewReader(dicomData))
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
		fmt.Printf("%s", respBytes)
		return nil
	}

	return nil
}
