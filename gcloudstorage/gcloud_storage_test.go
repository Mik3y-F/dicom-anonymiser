package gcpcloudstorage_test

import (
	"testing"

	gcloudstorage "gitlab.com/medical-research/dicom-deidentifier/gcloudstorage"
)

func TestNewGCloudStorage(t *testing.T) {
	tests := []struct {
		name    string
		want    *gcloudstorage.GCloudStorage
		wantErr bool
	}{
		{
			name:    "successfully created new gcloud storage",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gcloudstorage.NewGCloudStorage()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGCloudStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("GcloudStorage was not created")
			}
		})
	}
}
