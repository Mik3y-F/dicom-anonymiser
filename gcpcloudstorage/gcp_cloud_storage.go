package gcpcloudstorage

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

type CloudStorage struct {
	Client *storage.Client
}

func NewCloudStorage() (*CloudStorage, error) {

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error: %v", err)
	}

	store := &CloudStorage{
		Client: client,
	}

	return store, nil
}
