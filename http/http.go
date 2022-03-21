package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dicomdeidentifier "gitlab.com/medical-research/dicom-deidentifier"
)

// Generic HTTP metrics.
var (
	errorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dicomdeidentifier_http_error_count",
		Help: "Total number of errors by error code",
	}, []string{"code"})
)

// Client represents an HTTP client.
type Client struct {
	URL string
}

// NewClient returns a new instance of Client.
func NewClient(u string) *Client {
	return &Client{URL: u}
}

// newRequest returns a new HTTP request
// and sets the accept & content type headers to use JSON.
// func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
// 	// Build new request with base URL.
// 	req, err := http.NewRequest(method, c.URL+url, body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Default to JSON format.
// 	req.Header.Set("Accept", "application/json")
// 	req.Header.Set("Content-type", "application/json")

// 	return req, nil
// }

// Error prints & optionally logs an error message.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	// Extract error code & message.
	code, message := dicomdeidentifier.ErrorCode(err), dicomdeidentifier.ErrorMessage(err)

	// Track metrics by code.
	errorCount.WithLabelValues(code).Inc()

	// Log & report internal errors.
	if code == dicomdeidentifier.EINTERNAL {
		dicomdeidentifier.ReportError(r.Context(), err, r)
		LogError(r, err)
	}

	// Print user message to response based on reqeust accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(ErrorStatusCode(code))
		err = json.NewEncoder(w).Encode(&ErrorResponse{Error: message})
		if err != nil {
			LogError(r, err)
		}
	}
}

// ErrorResponse represents a JSON structure for error output.
type ErrorResponse struct {
	Error string `json:"error"`
}

// parseResponseError parses an JSON-formatted error response.
func parseResponseError(resp *http.Response) error {
	defer resp.Body.Close()

	// Read the response body so we can reuse it for the error message if it
	// fails to decode as JSON.
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse JSON formatted error response.
	// If not JSON, use the response body as the error message.
	var errorResponse ErrorResponse
	if err := json.Unmarshal(buf, &errorResponse); err != nil {
		message := strings.TrimSpace(string(buf))
		if message == "" {
			message = "Empty response from server."
		}
		return dicomdeidentifier.Errorf(FromErrorStatusCode(resp.StatusCode), message)
	}
	return dicomdeidentifier.Errorf(FromErrorStatusCode(resp.StatusCode), errorResponse.Error)
}

// LogError logs an error with the HTTP route information.
func LogError(r *http.Request, err error) {
	log.Printf("[http] error: %s %s: %s", r.Method, r.URL.Path, err)
}

// lookup of application error codes to HTTP status codes.
var codes = map[string]int{
	dicomdeidentifier.ECONFLICT:       http.StatusConflict,
	dicomdeidentifier.EINVALID:        http.StatusBadRequest,
	dicomdeidentifier.ENOTFOUND:       http.StatusNotFound,
	dicomdeidentifier.ENOTIMPLEMENTED: http.StatusNotImplemented,
	dicomdeidentifier.EUNAUTHORIZED:   http.StatusUnauthorized,
	dicomdeidentifier.EINTERNAL:       http.StatusInternalServerError,
}

// ErrorStatusCode returns the associated HTTP status code for a dicomdeidentifier error code.
func ErrorStatusCode(code string) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}

// FromErrorStatusCode returns the associated dicomdeidentifier code for an HTTP status code.
func FromErrorStatusCode(code int) string {
	for k, v := range codes {
		if v == code {
			return k
		}
	}
	return dicomdeidentifier.EINTERNAL
}

// WriteJSONResponse writes the content supplied via the `source` parameter to
// the supplied http ResponseWriter. The response is returned with the indicated
// status.
func WriteJSONResponse(w http.ResponseWriter, source interface{}, status int) {
	w.WriteHeader(status) // must come first...otherwise the first call to Write... sets an implicit 200
	content, errMap := json.Marshal(source)
	if errMap != nil {
		msg := fmt.Sprintf("error when marshalling %#v to JSON bytes: %#v", source, errMap)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, errMap = w.Write(content)
	if errMap != nil {
		msg := fmt.Sprintf(
			"error when writing JSON %s to http.ResponseWriter: %#v", string(content), errMap)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

}
