package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
)

// DefaultConfigPath is the default path to load configuration from
const DefaultConfigPath = "/etc/easyoss/config.yaml"

// Config holds all application configuration
type Config struct {
	S3Port   int    `yaml:"s3_port" env:"EASYOSS_S3_PORT" env-default:"9000" env-description:"S3 API port"`
	WebPort  int    `yaml:"web_port" env:"EASYOSS_WEB_PORT" env-default:"9001" env-description:"Web UI port"`
	DataPath string `yaml:"data_path" env:"EASYOSS_DATA_PATH" env-default:"./data" env-description:"Data storage path"`
	MetaPath string `yaml:"meta_path" env:"EASYOSS_META_PATH" env-default:"" env-description:"Metadata storage path (defaults to <data>/.meta)"`
}

// Load loads configuration from file and environment variables
// If configPath is empty, it tries to load from the default path
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	// If no config path specified, try to use default
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, load it
		if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else if configPath != DefaultConfigPath {
		// If a custom config path was specified but doesn't exist, return error
		return nil, fmt.Errorf("config file not found: %s", configPath)
	} else {
		// Default config file doesn't exist, just load from environment variables
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, fmt.Errorf("failed to read environment variables: %w", err)
		}
	}

	// Resolve meta path default if not set
	if cfg.MetaPath == "" {
		cfg.MetaPath = filepath.Join(cfg.DataPath, ".meta")
	}

	return cfg, nil
}

// LoadFromArgs parses command line arguments and loads configuration
// Returns the config and a boolean indicating if version was requested
func LoadFromArgs() (*Config, bool, error) {
	var configPath string
	var showVersion bool

	flag.StringVar(&configPath, "c", "", "Path to configuration file (default: "+DefaultConfigPath+")")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	if showVersion {
		return nil, true, nil
	}

	cfg, err := Load(configPath)
	if err != nil {
		return nil, false, err
	}

	return cfg, false, nil
}
