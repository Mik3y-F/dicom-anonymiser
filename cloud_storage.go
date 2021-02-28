package dicomdeidentifier

// StorageObject represents a single instance of a cloud storage object
type CloudStorageObject struct {
	Name string `json:"object-name"`
}

type SignedBucketURL struct {
	URL    string `json:"url,omitempty"`
	Status string `json:"status,omitempty"`
}

// CloudStorageBucket represents a single instance of a cloud storage bucket
type CloudStorageBucket struct {
	Name string `json:"name"`
}

// CloudStorageBucketService is an impentable interface with various operations that can be performed on a storage bucket
type CloudStorageService interface {

	// Generates a presigned bucket URL with limited possible operations for a limited period of time
	GeneratePresignedBucketURL(bucket *CloudStorageBucket, object *CloudStorageObject, method string) (*SignedBucketURL, error)
}
