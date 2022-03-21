package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	dcmd "gitlab.com/medical-research/dicom-deidentifier"
)

type BaseRequest struct {
	Subject string `json:"subject"`
}

type GenerateDicomURLResponsePayload struct {
	*dcmd.SignedBucketURL
	URLSigningStatus string `json:"url-signing-status"`
}

func (s *Server) wsStartDeidentification(w http.ResponseWriter, r *http.Request) {
	s.WebSocketUpgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := s.WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	s.wsReader(w, r, ws)
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func (s *Server) wsReader(w http.ResponseWriter, r *http.Request, conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(string(p))
			Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "streamed messages could not be read: %v", err))
			return
		}

		fmt.Println(string(p))

		req := &BaseRequest{}

		err = json.Unmarshal(p, req)
		if err != nil {
			Error(w, r, dcmd.Errorf(dcmd.EINVALID, "invalid request"))
			return
		}

		switch req.Subject {
		case "get-signed-url":
			bucketName := dcmd.MustGetEnvVar(StorageBucketName)
			bucket := &dcmd.CloudStorageBucket{
				Name: bucketName,
			}

			object := &dcmd.CloudStorageObject{}

			err := s.GenerateRequestMessage(object, p)
			if err != nil {
				err = conn.WriteMessage(1, []byte("{\"error\": upload failed}"))
				if err != nil {
					Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "error could not be streamed to client"))
					return
				}
				Error(w, r, dcmd.Errorf(dcmd.EINVALID, "invalid request"))
				return
			}

			// hardcoded the allowed method operation for security purposes
			// wanted the least amount of permissions to be allowed/alloweable
			signedURL, err := s.CloudStorageService.GeneratePresignedBucketURL(bucket, object, "PUT")
			if err != nil {
				err = conn.WriteMessage(1, []byte("{\"error\": upload failed}"))
				if err != nil {
					Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "error could not be streamed to client"))
					return
				}
				Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "signed URL could not be generated"))
				return
			}

			resp := &GenerateDicomURLResponsePayload{}
			resp.SignedBucketURL = signedURL
			resp.URLSigningStatus = "success"

			err = conn.WriteJSON(resp)
			if err != nil {
				Error(w, r, dcmd.Errorf(dcmd.EINTERNAL, "signed URL could not be sent"))
				return
			}

		case "finished-image-upload":

		default:
			log.Println(p)
			return

		}

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}

	}
}

func (s *Server) GenerateRequestMessage(reqMsg interface{}, message []byte) error {
	err := json.Unmarshal(message, reqMsg)
	if err != nil {
		return fmt.Errorf("unable to generate request message: %v", err)
	}

	return nil
}
