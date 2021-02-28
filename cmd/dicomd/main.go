package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/rollbar/rollbar-go"
	dcmd "gitlab.com/medical-research/dicom-deidentifier"
	gcloudstorage "gitlab.com/medical-research/dicom-deidentifier/gcloudstorage"
	"gitlab.com/medical-research/dicom-deidentifier/healthcare"
	"gitlab.com/medical-research/dicom-deidentifier/http"
)

const (
	RollBarToken = "ROLLBAR_TOKEN"
	HTTPAddress  = "HTTP_ADDRESS"
	Domain       = "DOMAIN"
)

// Build version, injected during build.
// var (
// 	version string
// )

func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Instantiate a new type to represent our application.
	// This type lets us shared setup code with our end-to-end tests.
	m, err := NewMain(ctx)
	if err != nil {
		log.Panicf("new main could not be created: %v", err)
		os.Exit(1)
	}

	// Execute program.
	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		dcmd.ReportError(ctx, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up program.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

// Main represents the program.
type Main struct {
	// Configuration path and parsed config data.
	ConfigPath string

	// HTTP server for handling HTTP communication.
	// DicomAPI services are attached to it before running.
	HTTPServer   *http.Server
	DicomAPI     *healthcare.GoogleDicomAPI
	CloudStorage *gcloudstorage.GCloudStorage
}

// NewMain returns a new instance of Main.
func NewMain(ctx context.Context) (*Main, error) {

	dicomAPI, err := healthcare.NewDicomAPI(ctx)
	if err != nil {
		return nil, err
	}

	cloudStorage, err := gcloudstorage.NewGCloudStorage()
	if err != nil {
		return nil, err
	}

	return &Main{
		DicomAPI:     dicomAPI,
		CloudStorage: cloudStorage,
		HTTPServer:   http.NewServer(),
	}, nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Run executes the program. The configuration should already be set up before
// calling this function.
func (m *Main) Run(ctx context.Context) (err error) {
	// Initialize error tracking.
	rollbarToken := dcmd.MustGetEnvVar(RollBarToken)

	rollbar.SetToken(rollbarToken)
	rollbar.SetEnvironment("development")
	rollbar.SetServerRoot("gitlab.com/medical-research/dicom-deidentifier")
	log.Printf("rollbar error tracking enabled")

	// Instantiate DicomAPI-backed services.
	dicomService := healthcare.NewDicomService(m.DicomAPI)
	dicomStoreService := healthcare.NewDicomStoreService(m.DicomAPI)
	cloudStorageService := gcloudstorage.NewCloudStorageService(m.CloudStorage)

	// Copy configuration settings to the HTTP server.
	httpAddress := os.Getenv(HTTPAddress)
	domain := os.Getenv(Domain)

	m.HTTPServer.Addr = httpAddress
	m.HTTPServer.Domain = domain
	m.HTTPServer.DicomService = dicomService
	m.HTTPServer.DicomStoreService = dicomStoreService
	m.HTTPServer.CloudStorageService = cloudStorageService

	// Start the HTTP server.
	if err := m.HTTPServer.Open(); err != nil {
		return err
	}

	// If TLS enabled, redirect non-TLS connections to TLS.
	if m.HTTPServer.UseTLS() {
		go func() {
			log.Fatal(http.ListenAndServeTLSRedirect(domain))
		}()
	}

	// Enable internal debug endpoints.
	go func() { log.Fatal(http.ListenAndServeDebug()) }()

	port := dcmd.MustGetEnvVar(http.Port)
	log.Printf("running: url=%q debug=http://localhost:%s", m.HTTPServer.URL(), port)

	return nil
}
