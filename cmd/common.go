package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
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

func initCommonMantisClient(cmd *cobra.Command) error {
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
		Language: "pt_BR",
		BaseURL:  appConfig.BaseURL,
	}

	mantisClient = mantis.NewClient(authConfig, clientConfig)
	mantisCtx = context.Background()

	err = mantisClient.Auth.Authenticate(mantisCtx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	currentUserID = profile.UserID
	if cmd.Name() == "roles" {
		return nil
	}

	if appConfig.RoleID == 0 {
		if err := handleMissingRoleID(appConfig); err != nil {
			return err
		}
	} else {
		mantisClient.SetRoleID(strconv.Itoa(appConfig.RoleID))
	}

	if profileExists && profile.UserID != 0 {
		currentUser, err = mantisClient.Employee.GetEmployeeById(
			mantisCtx,
			profile.UserID,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to get employee information for '%d': %w",
				profile.UserID, err,
			)
		}
		currentUserID = currentUser.UserID
	}

	// Get the timesheet types
	timesheetTypes, err := mantisClient.Reference.GetReferenceTypes(mantisCtx, mantis.ReferenceTypeFilter{
		ColumnName: "MTS_TimesheetType",
		TableName:  "MTS_Timesheet",
	})
	initTimesheetLookups(timesheetTypes)

	return nil
}

func initTimesheetLookups(timesheetTypes []mantis.ReferenceType) {
	for _, timesheetType := range timesheetTypes {
		types.TimesheetTypeLookup[timesheetType.Name] = timesheetType.Value
		types.TimesheetTypeInverseLookup[timesheetType.Value] = timesheetType.Name
	}
}

func handleMissingRoleID(appConfig *config.Config) error {
	userRoles, err := showUserRoles(appConfig.Profiles[appConfig.DefaultProfile].UserID)
	if err != nil {
		return err
	}

	selectedRole, err := chooseUserRole(userRoles)
	if err != nil {
		return err
	}

	roleId := strconv.Itoa(int(selectedRole.ADRoleID))
	mantisClient.SetRoleID(roleId)

	appConfig.RoleID = int(selectedRole.ADRoleID)
	config.SaveConfig(appConfig)
	fmt.Printf("Role set to: %s (ID: %s)\n", selectedRole.Name, roleId)
	return nil
}

func typeCompletionFunc(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {

	var timesheetTypes []string
	for t := range types.TimesheetTypeLookup {
		if strings.HasPrefix(t, toComplete) {
			timesheetTypes = append(timesheetTypes, t)
		}
	}

	return timesheetTypes, cobra.ShellCompDirectiveNoFileComp
}

func projectAliasCompletionFunc(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.InitializeConfig()
	if err != nil {
		log.Printf("Error loading config for completion: %v", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	currentProfileName := cfg.DefaultProfile
	if profileName != "" {
		currentProfileName = profileName
	}

	profile, ok := cfg.Profiles[currentProfileName]
	if !ok {
		log.Printf(
			"Default profile '%s' not found for completion.",
			cfg.DefaultProfile,
		)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var aliases []string
	for alias := range profile.ProjectAliases {
		if strings.HasPrefix(alias, toComplete) {
			aliases = append(aliases, alias)
		}
	}
	return aliases, cobra.ShellCompDirectiveNoFileComp
}

func filterNameCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.InitializeConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	for name := range cfg.SavedFilters {
		if strings.HasPrefix(name, toComplete) {
			completions = append(completions, name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func filterDaysNameCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.InitializeConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	for name := range cfg.SavedDayFilters {
		if strings.HasPrefix(name, toComplete) {
			completions = append(completions, name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func timesheetIdCompletionFunc(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.InitializeConfig()
	if err != nil {
		log.Printf("Error loading config for completion: %v", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	currentProfileName := cfg.DefaultProfile
	if profileName != "" {
		currentProfileName = profileName
	}

	profile, ok := cfg.Profiles[currentProfileName]
	if !ok {
		log.Printf(
			"Default profile '%s' not found for completion.",
			cfg.DefaultProfile,
		)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	filename := fmt.Sprintf(cache.TimesheetsCacheFileName, profile.UserID)
	timesheets, err := cache.ReadFromCache[mantis.TimesheetsResponse](filename)

	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, ts := range timesheets {
		timesheetIDStr := strconv.Itoa(ts.TimesheetID)
		var comment string

		parsedDate, err := time.Parse("2006-01-02T15:04:05Z", ts.DateDoc)
		if err == nil {
			mes := meses[int(parsedDate.Month())-1]
			formattedDate := fmt.Sprintf("%d de %s", parsedDate.Day(), mes)
			comment = fmt.Sprintf("(%.2f) %s [%s]", ts.Quantity, ts.Description, formattedDate)
		} else {
			comment = ts.Description
		}

		completions = append(
			completions,
			fmt.Sprintf("%s\t%s", timesheetIDStr, comment),
		)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func getCurrentProfile(cfg *config.Config) (config.Profile, error) {
	name := cfg.DefaultProfile
	if profileName != "" {
		name = profileName
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return config.Profile{}, fmt.Errorf("profile '%s' not found", name)
	}
	return p, nil
}
