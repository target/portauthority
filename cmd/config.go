package cmd

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/target/portauthority/api"
	"github.com/target/portauthority/pkg/datastore"

	"gopkg.in/yaml.v2"
)

// ErrDatasourceNotLoaded is returned when the datasource variable in the
// configuration file is not loaded properly
var ErrDatasourceNotLoaded = errors.New("could not load configuration: no database source specified")

// File represents a YAML configuration file that namespaces all Port Authority
// configuration under the top-level "portauthority" key
type File struct {
	PortAuthority Config `yaml:"portauthority"`
}

// Config is the global configuration for an instance of Port Authority
type Config struct {
	Database datastore.BackendConfig
	API      *api.Config
}

// DefaultConfig is a configuration that can be used as a fallback value
func DefaultConfig() Config {
	return Config{
		Database: datastore.BackendConfig{
			Type: "pgsql",
		},
		API: &api.Config{
			Port:       8080,
			HealthPort: 8081,
			Timeout:    900 * time.Second,
		},
	}
}

// LoadConfig is a shortcut to open a file, read it, and generate a Config.
// It supports relative and absolute paths. Given "", it returns DefaultConfig.
func LoadConfig(path string) (config *Config, err error) {
	var cfgFile File
	cfgFile.PortAuthority = DefaultConfig()
	if path == "" {
		return &cfgFile.PortAuthority, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(d, &cfgFile)
	if err != nil {
		return
	}
	config = &cfgFile.PortAuthority

	return
}
