package dicom

// ImageFile represents a single instance of a DICOM Image
type ImageFile struct {
	Name string
	Path string
}

// Service is an impentable interface with various operations that can be performed on DICOM Images
type Service interface {
	StoreMultipleDICOMImageInstances(df []*ImageFile) error
	StoreDICOMImageInstance(df *ImageFile) error
	AnonymiseDICOMS(df []*ImageFile) error
	PurgeDICOMS() error
}
