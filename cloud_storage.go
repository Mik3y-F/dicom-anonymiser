package dicomdeidentifier

// StorageObject represents a single instance of a cloud storage object
type CloudStorageObject struct {
	Name      string          `json:"name"`
	SignedURL SignedBucketURL `json:"signedURL"`
}

type SignedBucketURL struct {
	URL string `json:"url"`
}

// CloudStorageBucket represents a single instance of a cloud storage bucket
type CloudStorageBucket struct {
	Name string `json:"name"`
}

// CloudStorageBucketService is an impentable interface with various operations that can be performed on a storage bucket
type CloudStorageService interface {

	// Generates a presigned bucket URL with limited possible operations for a limited period of time
	GeneratePresignedBucketURL(bucket *CloudStorageBucket, object *CloudStorageObject, serviceAccount, method string) (*SignedBucketURL, error)
}
