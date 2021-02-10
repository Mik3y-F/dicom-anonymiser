package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rollbar/rollbar-go"
	"gitlab.com/medical-research/dicom-deidentifier/gcpcloudstorage"
	"gitlab.com/medical-research/dicom-deidentifier/healthcare"
	"gitlab.com/medical-research/dicom-deidentifier/http"
)

// Build version, injected during build.
var (
	version string
)

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
		os.Exit(1) // TODO: Supposed to panic if New Main cannot be created
	}

	// Parse command line flags & load configuration.
	if err := m.ParseFlags(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

// Main represents the program.
type Main struct {
	// Configuration path and parsed config data.
	Config     Config
	ConfigPath string

	// HTTP server for handling HTTP communication.
	// DicomAPI services are attached to it before running.
	HTTPServer   *http.Server
	DicomAPI     *healthcare.DicomAPI
	CloudStorage *gcpcloudstorage.CloudStorage
}

// NewMain returns a new instance of Main.
func NewMain(ctx context.Context) (*Main, error) {

	dicomAPI, err := healthcare.NewDicomAPI(ctx)
	if err != nil {
		return nil, err
	}

	cloudStorage, err := gcpcloudstorage.NewCloudStorage()
	if err != nil {
		return nil, err
	}

	return &Main{
		Config:       DefaultConfig(),
		ConfigPath:   DefaultConfigPath,
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

// ParseFlags parses the command line arguments & loads the config.
//
// This exists separately from the Run() function so that we can skip it
// during end-to-end tests. Those tests will configure manually and call Run().
func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	// Our flag set is very simple. It only includes a config path.
	fs := flag.NewFlagSet("dicomd", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// The expand() function is here to automatically expand "~" to the user's
	// home directory. This is a common task as configuration files are typing
	// under the home directory during local development.
	configPath, err := expand(m.ConfigPath)
	if err != nil {
		return err
	}

	// Read our TOML formatted configuration file.
	config, err := ReadConfigFile(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", m.ConfigPath)
	} else if err != nil {
		return err
	}
	m.Config = config

	return nil
}

// Run executes the program. The configuration should already be set up before
// calling this function.
func (m *Main) Run(ctx context.Context) (err error) {
	// Initialize error tracking.
	if m.Config.Rollbar.Token != "" {
		rollbar.SetToken(m.Config.Rollbar.Token)
		rollbar.SetEnvironment("production")
		rollbar.SetCodeVersion(version)
		rollbar.SetServerRoot("gitlab.com/medical-research/dicom-deidentifier")
		log.Printf("rollbar error tracking enabled")
	}

	// Instantiate DicomAPI-backed services.
	dicomService := healthcare.NewDicomService(m.DicomAPI)
	dicomStoreService := healthcare.NewDicomStoreService(m.DicomAPI)
	cloudStorageService := gcpcloudstorage.NewCloudStorageService(m.CloudStorage)

	// Copy configuration settings to the HTTP server.
	m.HTTPServer.Addr = m.Config.HTTP.Addr
	m.HTTPServer.Domain = m.Config.HTTP.Domain
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
			log.Fatal(http.ListenAndServeTLSRedirect(m.Config.HTTP.Domain))
		}()
	}

	// Enable internal debug endpoints.
	go func() { log.Fatal(http.ListenAndServeDebug()) }()

	log.Printf("running: url=%q debug=http://localhost:6060", m.HTTPServer.URL())

	return nil
}

const (
	// DefaultConfigPath is the default path to the application configuration.
	DefaultConfigPath = "~/dicomd.conf"
)

// Config represents the CLI configuration file.
type Config struct {
	HTTP struct {
		Addr     string `toml:"addr"`
		Domain   string `toml:"domain"`
		HashKey  string `toml:"hash-key"`
		BlockKey string `toml:"block-key"`
	} `toml:"http"`

	GoogleAnalytics struct {
		MeasurementID string `toml:"measurement-id"`
	} `toml:"google-analytics"`

	Rollbar struct {
		Token string `toml:"token"`
	} `toml:"rollbar"`
}

// DefaultConfig returns a new instance of Config with defaults set.
func DefaultConfig() Config {
	var config Config
	return config
}

// ReadConfigFile unmarshals config from
func ReadConfigFile(filename string) (Config, error) {
	config := DefaultConfig()
	if buf, err := ioutil.ReadFile(filename); err != nil {
		return config, err
	} else if err := toml.Unmarshal(buf, &config); err != nil {
		return config, err
	}
	return config, nil
}

// expand returns path using tilde expansion. This means that a file path that
// begins with the "~" will be expanded to prefix the user's home directory.
func expand(path string) (string, error) {
	// Ignore if path has no leading tilde.
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	// Fetch the current user to determine the home path.
	u, err := user.Current()
	if err != nil {
		return path, err
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}
	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil
}
