package gcpcloudstorage_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"cloud.google.com/go/storage"
	dcmd "gitlab.com/medical-research/dicom-deidentifier"
	gcloudstorage "gitlab.com/medical-research/dicom-deidentifier/gcloudstorage"
)

const (
	testSignedURL = "https://test-signed-url-success.com"
)

var (
	testBucket = &dcmd.CloudStorageBucket{
		Name: "test-bucket",
	}

	testObject = &dcmd.CloudStorageObject{
		Name: "test-object",
	}
)

func mockSuccessfullyCreatedSignedURL(bucket, name string, opts *storage.SignedURLOptions) (string, error) {
	return testSignedURL, nil
}

func mockErrorOccurredCreatingSignedURL(bucket, name string, opts *storage.SignedURLOptions) (string, error) {
	return "", fmt.Errorf("no signed url generated")
}

func TestCloudStorageService_GeneratePresignedBucketURL(t *testing.T) {

	type fields struct {
		CloudStorage *gcloudstorage.GCloudStorage
	}
	type args struct {
		bucket *dcmd.CloudStorageBucket
		object *dcmd.CloudStorageObject
		method string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		setEnv         bool
		serviceAccount string
		want           *dcmd.SignedBucketURL
		wantErr        bool
	}{
		{
			name: "successfully generated presigned bucket url",
			fields: fields{
				CloudStorage: &gcloudstorage.GCloudStorage{
					SignedURL: mockSuccessfullyCreatedSignedURL,
				},
			},
			args: args{
				bucket: testBucket,
				object: testObject,
			},
			setEnv: false,
			want: &dcmd.SignedBucketURL{
				URL: testSignedURL,
			},
			wantErr: false,
		},
		{
			name: "unsuccessful generation of presigned bucket url",
			fields: fields{
				CloudStorage: &gcloudstorage.GCloudStorage{
					SignedURL: mockErrorOccurredCreatingSignedURL,
				},
			},
			setEnv: false,
			args: args{
				bucket: testBucket,
				object: testObject,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "non-existent service account supplied",
			fields: fields{
				CloudStorage: &gcloudstorage.GCloudStorage{
					SignedURL: mockErrorOccurredCreatingSignedURL,
				},
			},
			setEnv:         true,
			serviceAccount: "/non-existant-service-account.json",
			args: args{
				bucket: testBucket,
				object: testObject,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentServiceAccount := os.Getenv(gcloudstorage.ServiceAccount)
			if tt.setEnv == true {
				err := os.Setenv(gcloudstorage.ServiceAccount, tt.serviceAccount)
				if err != nil {
					t.Errorf("test environment variable could not be set: %v", err)
				}
			}
			s := &gcloudstorage.CloudStorageService{
				GCloudStorage: tt.fields.CloudStorage,
			}
			got, err := s.GeneratePresignedBucketURL(tt.args.bucket, tt.args.object, tt.args.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloudStorageService.GeneratePresignedBucketURL() error = %v, wantErr %v", err, tt.wantErr)
				if tt.setEnv == true {
					os.Setenv(gcloudstorage.ServiceAccount, currentServiceAccount)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloudStorageService.GeneratePresignedBucketURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCloudStorageService(t *testing.T) {
	type args struct {
		gcloudStorage *gcloudstorage.GCloudStorage
	}
	tests := []struct {
		name string
		args args
		want *gcloudstorage.CloudStorageService
	}{
		{
			name: "successfully created new cloud storage service",
			args: args{
				gcloudStorage: nil,
			},
			want: &gcloudstorage.CloudStorageService{
				GCloudStorage: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gcloudstorage.NewCloudStorageService(tt.args.gcloudStorage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCloudStorageService() = %v, want %v", got, tt.want)
			}
		})
	}
}
