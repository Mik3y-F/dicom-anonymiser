package http

import (
	"encoding/json"
	"net/http"

	dcmd "gitlab.com/medical-research/dicom-deidentifier"
)

// handleGetPresignedBucketURL handles the "POST /get_presigned_url" route.
func (s *Server) handleGetPresignedBucketURL(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	bucketName := dcmd.MustGetEnvVar(StorageBucketName)
	bucket := &dcmd.CloudStorageBucket{
		Name: bucketName,
	}

	object := &dcmd.CloudStorageObject{}
	err := json.NewDecoder(r.Body).Decode(object)
	if err != nil {
		Error(w, r, dcmd.Errorf(dcmd.EINVALID, "invalid request"))
		return
	}

	// hardcoded the allowed method operation for security purposes
	// wanted the least amount of permissions to be allowed/alloweable
	signedURL, err := s.CloudStorageService.GeneratePresignedBucketURL(bucket, object, "POST")
	if err != nil {
		Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "signed URL could not be generated"))
		return
	}

	WriteJSONResponse(w, &signedURL, 200)
}

// handleStartAnonymisation handles the "POST /start_anonymisation" route.
func (s *Server) handleStartAnonymisation(w http.ResponseWriter, r *http.Request) {
	// TODO: Begin anonymisation
	// TODO: Export Anonymised DICOMS to Bucket
	// TODO: Send report to client(frontend)
	// TODO: Send presigned link to the patient record system
}
