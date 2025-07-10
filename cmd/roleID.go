package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var modifyProfile bool

func init() {
	rolesCmd.Flags().BoolVarP(
		&modifyProfile,
		"modify",
		"m",
		false,
		"Modify current profile configuration to the choosen roleID",
	)

	rootCmd.AddCommand(rolesCmd)
}

var rolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "Get user roles",
	Long: "Searchs for the roles in Mantis by a given userID." +
		"Useful for changing it in the configuration profile",

	Run: func(cmd *cobra.Command, args []string) {
		userRoles, err := showUserRoles(currentUserID)
		if err != nil {
			log.Fatalf("Error getting user roles: %v", err)
		}

		if !modifyProfile {
			return
		}

		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}

		profile, ok := appConfig.Profiles[currentProfileName]
		if !ok {
			log.Fatalf("Profile '%s' not found in config", currentProfileName)
		}
		selectedRole, err := chooseUserRole(userRoles)
		if err != nil {
			log.Fatalf("Error getting user role: %v", err)
		}

		roleId := strconv.Itoa(int(selectedRole.ADRoleID))
		mantisClient.SetRoleID(roleId)
		profile.RoleID = int(selectedRole.ADRoleID)

		log.Println(currentProfileName, profile)
		appConfig.Profiles[currentProfileName] = profile
		log.Println(appConfig)
		err = config.SaveConfig(appConfig)

		if err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		fmt.Printf(
			"RoleID '%s' (%d) saved in config\n",
			selectedRole.Name,
			selectedRole.ADRoleID,
		)
	},
}

func showUserRoles(userId int) ([]mantis.UserRole, error) {
	userRoles, err := mantisClient.GetUserRoles(mantisCtx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user roles: %w", err)
	}

	if len(userRoles) == 0 {
		return nil, fmt.Errorf("no roles available for user %d", userId)
	}

	for i, role := range userRoles {
		fmt.Printf("%d. %s (ID: %d)\n", i+1, role.Name, role.ADRoleID)
	}

	return userRoles, nil
}

func chooseUserRole(userRoles []mantis.UserRole) (mantis.UserRole, error) {
	var choiceStr string
	var chosenIndex int

	for {
		fmt.Print("Enter the number of your chosen role: ")
		_, err := fmt.Scanln(&choiceStr)
		if err != nil {
			return mantis.UserRole{}, fmt.Errorf("error reading choice: %w", err)
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

	return selectedRole, nil
}
