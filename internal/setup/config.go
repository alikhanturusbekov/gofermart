package setup

import (
	"errors"
	"flag"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	configFilePath = "configs/config.yaml"
)

type Config struct {
	RunAddress              string `yaml:"run_address"`
	DatabaseURI             string `yaml:"database_uri"`
	AccrualSystemAddress    string `yaml:"accrual_system_address"`
	AuthenticationSecretKey string `yaml:"authentication_secret_key"`
}

// LoadConfig loads application configuration
func LoadConfig() (*Config, error) {
	config := Config{}

	// Load configuration file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Flag configurations overwrite yaml configurations
	flag.StringVar(&config.RunAddress, "a", config.RunAddress, "HTTP server run address")
	flag.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "Database connection string")
	flag.StringVar(&config.AccrualSystemAddress, "r", config.DatabaseURI, "Accrual system address")
	flag.Parse()

	// Validate the configs are valid
	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// validate validates that the config is acceptable
func (c *Config) validate() error {
	if (c.RunAddress == "") || (c.DatabaseURI == "") {
		return errors.New("must specify all configs: run_address, database_dsn")
	}

	return nil
}
