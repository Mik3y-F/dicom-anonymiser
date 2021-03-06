package gcpcloudstorage

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"cloud.google.com/go/storage"
	dcmd "gitlab.com/medical-research/dicom-deidentifier"
	"golang.org/x/oauth2/google"
)

const (
	// only used for the signed url generation
	ServiceAccount = "SERVICE_ACCOUNT"
)

type SignedURL func(bucket, name string, opts *storage.SignedURLOptions) (string, error)

type GCloudStorage struct {
	Client    *storage.Client
	SignedURL SignedURL
}

func NewGCloudStorage() (*GCloudStorage, error) {

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error: %v", err)
	}

	store := &GCloudStorage{
		Client:    client,
		SignedURL: storage.SignedURL,
	}

	return store, nil
}

// Ensure service implements interface.
var _ dcmd.CloudStorageService = (*CloudStorageService)(nil)

// CloudStorageService represents a service for managing CloudStorages
type CloudStorageService struct {
	GCloudStorage *GCloudStorage
}

// NewCloudStorageService returns a new instance of CloudStorageService
func NewCloudStorageService(gcloudStorage *GCloudStorage) *CloudStorageService {
	return &CloudStorageService{
		GCloudStorage: gcloudStorage,
	}
}

//GeneratePresignedBucketURL Generates a presigned bucket URL with limited possible operations for a limited period of time
func (s *CloudStorageService) GeneratePresignedBucketURL(bucket *dcmd.CloudStorageBucket, object *dcmd.CloudStorageObject, method string) (*dcmd.SignedBucketURL, error) {

	serviceAccount := dcmd.MustGetEnvVar(ServiceAccount)
	jsonKey, err := ioutil.ReadFile(serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile: %v", err)
	}
	conf, err := google.JWTConfigFromJSON(jsonKey)
	if err != nil {
		return nil, fmt.Errorf("google.JWTConfigFromJSON: %v", err)
	}
	opts := &storage.SignedURLOptions{
		Scheme: storage.SigningSchemeV4,
		Method: method,
		Headers: []string{
			"Content-Type:application/octet-stream",
		},
		GoogleAccessID: conf.Email,
		PrivateKey:     conf.PrivateKey,
		Expires:        time.Now().Add(15 * time.Minute),
	}
	u, err := s.GCloudStorage.SignedURL(bucket.Name, object.Name, opts)
	if err != nil {
		return nil, fmt.Errorf("storage.SignedURL: %v", err)
	}
	return &dcmd.SignedBucketURL{URL: u}, nil
}
