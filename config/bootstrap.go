package config

import (
	"fmt"
)

func RunBootstrap(cfg *Config) error {
	fmt.Println("IntraCLI is not configured yet.")
	fmt.Println("Let's create an initial profile.")

	var profileName string
	fmt.Print("Profile name [default]: ")
	fmt.Scanln(&profileName)
	if profileName == "" {
		profileName = "default"
	}

	var baseURL string
	fmt.Print("Mantis Base URL(https://mantis-br.nttdata-solutions.com): ")
	fmt.Scanln(&baseURL)

	var userID int
	fmt.Print("User ID (optional, press enter to auto-detect): ")
	fmt.Scanln(&userID)

	cfg.BaseURL = baseURL
	cfg.DefaultProfile = profileName
	cfg.Profiles = map[string]Profile{
		profileName: {
			UserID:         userID,
			DailyJourney:   8,
			ProjectAliases: map[string]ProjectAlias{},
		},
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println("Configuration created.")

	return nil
}
