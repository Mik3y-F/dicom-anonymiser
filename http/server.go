package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"path"
	"strings"
	"time"

	dicomdeidentifier "gitlab.com/medical-research/dicom-deidentifier"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
)

// Generic HTTP metrics.
var (
	requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dicom_deidentifier_http_request_count",
		Help: "Total number of requests by route",
	}, []string{"method", "path"})

	requestSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dicom_deidentifier_http_request_seconds",
		Help: "Total amount of request time by route, in seconds",
	}, []string{"method", "path"})
)

// ShutdownTimeout is the time given for outstanding requests to finish before shutdown.
const ShutdownTimeout = 1 * time.Second

// Server represents an HTTP server. It is meant to wrap all HTTP functionality
// used by the application so that dependent packages (such as cmd/wtfd) do not
// need to reference the "net/http" package at all.
type Server struct {
	ln     net.Listener
	server *http.Server
	router *mux.Router

	// Bind address & domain for the server's listener.
	// If domain is specified, server is run on TLS using acme/autocert.
	Addr   string
	Domain string

	// Servics used by the various HTTP routes.
	DicomStoreService dicomdeidentifier.DicomStoreService
	DicomService      dicomdeidentifier.DicomService
}

// NewServer returns a new instance of Server.
func NewServer() *Server {
	// Create a new server that wraps the net/http server & add a gorilla router.
	s := &Server{
		server: &http.Server{},
		router: mux.NewRouter(),
	}

	// Report panics to external service.
	s.router.Use(reportPanic)

	// Our router is wrapped by another function handler to perform some
	// middleware-like tasks that cannot be performed by actual middleware.
	// This includes changing route paths for JSON endpoints & overridding methods.
	s.server.Handler = http.HandlerFunc(s.serveHTTP)

	// Setup a base router that excludes asset handling.
	router := s.router.PathPrefix("/").Subrouter()
	router.Use(trackMetrics)

	// Handle authentication check within handler function for dicom deidntification endpoint.
	router.HandleFunc("/deidentify_dicoms", s.handleDeidentifyDicoms).Methods("POST")

	return s
}

// UseTLS returns true if the cert & key file are specified.
func (s *Server) UseTLS() bool {
	return s.Domain != ""
}

// Scheme returns the URL scheme for the server.
func (s *Server) Scheme() string {
	if s.UseTLS() {
		return "https"
	}
	return "http"
}

// Port returns the TCP port for the running server.
// This is useful in tests where we allocate a random port by using ":0".
func (s *Server) Port() int {
	if s.ln == nil {
		return 0
	}
	return s.ln.Addr().(*net.TCPAddr).Port
}

// URL returns the local base URL of the running server.
func (s *Server) URL() string {
	scheme, port := s.Scheme(), s.Port()

	// Use localhost unless a domain is specified.
	domain := "localhost"
	if s.Domain != "" {
		domain = s.Domain
	}

	// Return without port if using standard ports.
	if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
		return fmt.Sprintf("%s://%s", s.Scheme(), domain)
	}
	return fmt.Sprintf("%s://%s:%d", s.Scheme(), domain, s.Port())
}

// Open validates the server options and begins listening on the bind address.
func (s *Server) Open() (err error) {

	// Open a listener on our bind address.
	if s.Domain != "" {
		s.ln = autocert.NewListener(s.Domain)
	} else {
		if s.ln, err = net.Listen("tcp", s.Addr); err != nil {
			return err
		}
	}

	// Begin serving requests on the listener. We use Serve() instead of
	// ListenAndServe() because it allows us to check for listen errors (such
	// as trying to use an already open port) synchronously.
	go s.server.Serve(s.ln)

	return nil
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// Override method for forms passing "_method" value.
	if r.Method == http.MethodPost {
		switch v := r.PostFormValue("_method"); v {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			r.Method = v
		}
	}

	// Override content-type for certain extensions.
	// This allows us to easily cURL API endpoints with a ".json" or ".csv"
	// extension instead of having to explicitly set Content-type & Accept headers.
	// The extensions are removed so they don't appear in the routes.
	switch ext := path.Ext(r.URL.Path); ext {
	case ".json":
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-type", "application/json")
		r.URL.Path = strings.TrimSuffix(r.URL.Path, ext)
	case ".csv":
		r.Header.Set("Accept", "text/csv")
		r.URL.Path = strings.TrimSuffix(r.URL.Path, ext)
	}

	// Delegate remaining HTTP handling to the gorilla router.
	s.router.ServeHTTP(w, r)
}

// trackMetrics is middleware for tracking the request count and timing per route.
func trackMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtain path template & start time of request.
		t := time.Now()
		tmpl := requestPathTemplate(r)

		// Delegate to next handler in middleware chain.
		next.ServeHTTP(w, r)

		// Track total time unless it is the WebSocket endpoint for events.
		if tmpl != "" {
			requestCount.WithLabelValues(r.Method, tmpl).Inc()
			requestSeconds.WithLabelValues(r.Method, tmpl).Add(float64(time.Since(t).Seconds()))
		}
	})
}

// requestPathTemplate returns the route path template for r.
func requestPathTemplate(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return ""
	}
	tmpl, _ := route.GetPathTemplate()
	return tmpl
}

// reportPanic is middleware for catching panics and reporting them.
func reportPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				dicomdeidentifier.ReportPanic(err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// handleIndex handles the "POST /deidentify_dicom" route. It displays a dashboard with the
// user's dials, recently updated membership values, & a chart.
func (s *Server) handleDeidentifyDicoms(w http.ResponseWriter, r *http.Request) {
	// TODO: Unzip Uploaded zip file
	// TODO: Create Two DicomStores (Source & Destination)
	// TODO: Create DicomInstances in the Source Dicom store
	// TODO: Deidentify Dicoms in the Source Dicom Store
	// TODO: Retreive the Deidentified Dicom instances stored in the Destination Store
	// TODO: Upload the Deidentified Dicom zip file to another server
}

// handleVersion displays the deployed version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(dicomdeidentifier.Version))
}

// handleVersion displays the deployed commit.
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(dicomdeidentifier.Commit))
}

// ListenAndServeTLSRedirect runs an HTTP server on port 80 to redirect users
// to the TLS-enabled port 443 server.
func ListenAndServeTLSRedirect(domain string) error {
	return http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+domain, http.StatusFound)
	}))
}

// ListenAndServeDebug runs an HTTP server with /debug endpoints (e.g. pprof, vars).
func ListenAndServeDebug() error {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":6060", h)
}
