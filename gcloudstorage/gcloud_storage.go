package gcpcloudstorage

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
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
