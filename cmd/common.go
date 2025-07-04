package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var (
	mantisClient  *mantis.Client
	mantisCtx     context.Context
	currentUser   mantis.Employee
	currentUserID int
	appConfig     *config.Config
)

var (
	timesheetTypeLookup = map[string]string{
		"Normal":          "N",
		"OnDuty":          "D",
		"OnDutyOvertime":  "A",
		"Overtime":        "S",
		"OvertimeClosing": "C",
		"OnNotice":        "O",
		"Retroactive":     "R",
	}

	timesheetTypeInverseLookup = map[string]string{
		"R": "Retroactive",
		"O": "OnNotice",
		"C": "OvertimeClosing",
		"A": "OnDutyOvertime",
		"N": "Normal",
		"D": "OnDuty",
		"S": "Overtime",
	}
)

func initCommonMantisClient(_ *cobra.Command) error {
	var err error

	appConfig, err = config.InitializeConfig()
	if err != nil {
		fmt.Printf("Fatal error during config initialization: %v\n", err)
		os.Exit(1)
	}

	currentProfileName := appConfig.DefaultProfile
	if profileName != "" {
		currentProfileName = profileName
	}

	profile, profileExists := appConfig.Profiles[currentProfileName]

	username := os.Getenv("MANTIS_USERNAME")
	password := os.Getenv("MANTIS_PASSWORD")
	if username == "" || password == "" {
		fmt.Print("Username: ")
		fmt.Scanln(&username)
		fmt.Print("Password: ")
		fmt.Scanln(&password)
	}

	authConfig := mantis.AuthConfig{
		Username:     username,
		Password:     password,
		ClientID:     "api.oauth2-client.129d054eed33d25e3b6a714ca101f3b9",
		ClientSecret: "4eb3a90960f666799cb75ab8a68f1d5c",
	}

	clientConfig := &mantis.ClientConfig{
		RoleID:   "1000333",
		Language: "pt_BR",
	}

	mantisClient = mantis.NewClient(authConfig, clientConfig)
	mantisCtx = context.Background()

	err = mantisClient.Auth.Authenticate(mantisCtx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if profileExists && profile.UserID != 0 {
		currentUser, err = mantisClient.Employee.GetEmployeeById(mantisCtx, profile.UserID)
		if err != nil {
			return fmt.Errorf("failed to get employee information for '%d': %w", profile.UserID, err)
		}
		currentUserID = currentUser.UserID
	}

	return nil
}
