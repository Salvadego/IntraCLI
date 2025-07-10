package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		RoleID:   strconv.Itoa(profile.RoleID),
		Language: "pt_BR",
	}

	mantisClient = mantis.NewClient(authConfig, clientConfig)
	mantisCtx = context.Background()

	err = mantisClient.Auth.Authenticate(mantisCtx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if profile.RoleID == 0 {
		if err := handleMissingRoleID(profile); err != nil {
			return err
		}
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

func handleMissingRoleID(profile config.Profile) error {
	userRoles, err := mantisClient.GetUserRoles(mantisCtx, profile.UserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user roles: %w", err)
	}

	if len(userRoles) == 0 {
		return fmt.Errorf("no roles available for user %d", profile.UserID)
	}

	fmt.Println("\nPlease choose your role:")
	for i, role := range userRoles {
		fmt.Printf("%d. %s (ID: %d)\n", i+1, role.Name, role.ADRoleID)
	}

	var choiceStr string
	var chosenIndex int
	for {
		fmt.Print("Enter the number of your chosen role: ")
		_, err := fmt.Scanln(&choiceStr)
		if err != nil {
			return fmt.Errorf("error reading choice: %w", err)
		}

		choiceStr = strings.TrimSpace(choiceStr)
		chosenIndex, err = strconv.Atoi(choiceStr)
		isInvalidChoice := err != nil ||
			chosenIndex < 1 ||
			chosenIndex > len(userRoles)

		if isInvalidChoice {
			fmt.Printf(
				"Invalid choice. Please enter a number between 1 and %d.\n",
				len(userRoles),
			)
			continue
		}
		break
	}

	selectedRole := userRoles[chosenIndex-1]
	roleId := strconv.Itoa(int(selectedRole.ADRoleID))
	mantisClient.SetRoleID(roleId)
	fmt.Printf("Role set to: %s (ID: %s)\n", selectedRole.Name, roleId)
	return nil
}
