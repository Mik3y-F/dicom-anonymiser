package dicomdeidentifier

import "context"

// DicomStore represents a single instance of a Dicom Store
// a single Dicom store holds multiple Dicom instances
type DicomStore struct {
	StoreID string
}

// DicomStoreService is an impentable interface with various operations that can be performed on DICOM Images
type DicomStoreService interface {

	// Creates special storage abstractions in the cloud known as dicom stores
	// The dicom stores will hold the various dicom instances created
	CreateDicomStore(ctx context.Context, storeID string) (*DicomStore, error)

	// Deletes an existing dicom store
	DeleteDicomStore(ctx context.Context, storeID string) error

	// Generates a unique Dicom store name
	GenerateDicomStoreID(ctx context.Context) (string, error)

	// Lists all dicom stores created
	GetDicomStoreList(ctx context.Context) ([]*DicomStore, error)

	// Strips the P.I.I(Personally Identifiable Information) embedded in the dicom instances
	// Deidentified dicom instances will be stored in the destinationDicomStoreProvided
	DeidentifyDicomStore(ctx context.Context, sourceDicomStore, destinationDicomStore *DicomStore) error

	// Imports Dicom Instances from GCS
	ImportDICOMInstance(dicomStoreID, contentURI string) error
	// Exports Dicom Instances to GCS
	ExportDICOMInstance(dicomStoreID, gcsDestination string) error
}
