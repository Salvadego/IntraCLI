package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Salvadego/IntraCLI/types"
	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".config"
	appName    = "intracli"
	configFile = "config.yaml"
)

type Config struct {
	DefaultProfile  string                           `yaml:"defaultProfile"`
	Profiles        map[string]Profile               `yaml:"profiles"`
	RoleID          int                              `yaml:"roleID"`
	BaseURL         string                           `yaml:"baseURL"`
	SavedFilters    map[string]types.TimesheetFilter `yaml:"savedFilters"`
	SavedDayFilters map[string]types.DailyFilter     `yaml:"savedDayFilters"`
}

type Profile struct {
	EmployeeName   string                  `yaml:"employeeName"`
	UserID         int                     `yaml:"userID"`
	Email          string                  `yaml:"email"`
	EmployeeCode   int                     `yaml:"employeeCode"`
	DailyJourney   float64                 `yaml:"dailyJourney"`
	ProjectAliases map[string]ProjectAlias `yaml:"projectAliases"`
}

type ProjectAlias struct {
	SalesOrder     int  `yaml:"salesOrder"`
	SalesOrderLine int  `yaml:"salesOrderLine"`
	NeedsTicket    bool `yaml:"needsTicket"`
}

func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, configDir, appName, configFile), nil
}

func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found at %s. Please create it.", configPath)
		}
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration file: %w", err)
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	return nil
}

func InitializeConfig() (*Config, error) {
	cfg, err := LoadConfig()

	if err != nil {
		cfgDir, err := GetConfigPath()
		if err != nil {
			return nil, err
		}

		fmt.Printf("Configuration file not found. Creating a default config at %s...\n", cfgDir)

		defaultCfg := &Config{
			DefaultProfile: "",
			Profiles:       make(map[string]Profile),
		}

		saveErr := SaveConfig(defaultCfg)
		if saveErr != nil {
			return nil, fmt.Errorf("failed to save default config: %w", saveErr)
		}
		fmt.Println("Default config created successfully.")
		return defaultCfg, nil
	}

	return cfg, nil
}
