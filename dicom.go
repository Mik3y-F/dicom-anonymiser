package dicomdeidentifier

import "context"

// Dicom represents a single instance of a DICOM Image
type Dicom struct {
	Name string
	Path string
}

// DicomService is an impentable interface with various operations that can be performed on DICOM Images
type DicomService interface {

	// Creates and stores dicom instances in the cloud for further cloud operations
	// The dicom instances will be stored in special storage abstractions known as dicom stores
	CreateDicomInstances(ctx context.Context, dicomStore DicomStore, dicoms ...Dicom) error
}
