package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/Salvadego/IntraCLI/config"

	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var aliasName string
var projectNumber int

func init() {
	listProjectsCmd.Flags().StringVarP(&aliasName, "alias", "a", "", "Create an alias for a specific project")
	listProjectsCmd.Flags().IntVarP(&projectNumber, "project-number", "n", 0, "Project number to associate the alias with (required with --alias)")

	listProjectsCmd.RegisterFlagCompletionFunc("project-number", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

		if profile.UserID != 0 {
			currentUser, err = mantisClient.Employee.GetEmployeeById(
				mantisCtx,
				profile.UserID,
			)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			currentUserID = currentUser.UserID
		}

		projects, err := mantisClient.Timesheet.GetProjectTimesheets(mantisCtx, currentUser.EmployeeCode)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var suggestions []string
		for _, p := range projects {
			s := fmt.Sprintf("%d\t%s", p.ProjectNumber, p.ProjectTitle)
			if strings.HasPrefix(s, toComplete) {
				suggestions = append(suggestions, s)
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(listProjectsCmd)
}

var listProjectsCmd = &cobra.Command{
	Use:   "list-projects",
	Short: "List projects assigned to your profile",
	Long:  `Retrieves and displays all projects assigned to the employee configured in your profile, and optionally adds a project alias.`,
	Run: func(cmd *cobra.Command, args []string) {
		if currentUser.EmployeeCode == 0 {
			log.Fatalf("Employee code not found. Please run 'intracli search-employee' to update your profile.")
		}

		projects, err := mantisClient.Timesheet.GetProjectTimesheets(mantisCtx, currentUser.EmployeeCode)
		if err != nil {
			log.Fatalf("Error getting projects: %v", err)
		}

		if aliasName != "" {
			if projectNumber == 0 {
				log.Fatal("When using --alias, you must also provide --project-number.")
			}

			var selectedProject *mantis.ProjectTimesheet
			for _, p := range projects {
				if p.ProjectNumber == projectNumber {
					selectedProject = &p
					break
				}
			}

			if selectedProject == nil {
				log.Fatalf("Project number %d not found in current project list", projectNumber)
			}

			currentProfileName := appConfig.DefaultProfile
			if profileName != "" {
				currentProfileName = profileName
			}

			profile, ok := appConfig.Profiles[currentProfileName]
			if !ok {
				log.Fatalf("Profile '%s' not found in config", currentProfileName)
			}

			newAlias := config.ProjectAlias{
				SalesOrder:     selectedProject.ProjectNumber,
				SalesOrderLine: selectedProject.EmployeeLineNumber,
				NeedsTicket:    selectedProject.ProjectNeedTicket,
			}

			if profile.ProjectAliases == nil {
				profile.ProjectAliases = map[string]config.ProjectAlias{}
			}
			profile.ProjectAliases[aliasName] = newAlias

			appConfig.Profiles[currentProfileName] = profile

			err = config.SaveConfig(appConfig)
			if err != nil {
				log.Fatalf("Failed to save config: %v", err)
			}

			fmt.Printf("Alias '%s' saved for project %d (%s).\n", aliasName, selectedProject.ProjectNumber, selectedProject.ProjectTitle)
			return
		}

		if len(projects) == 0 {
			fmt.Println("No projects found for the current employee.")
			return
		}

		fmt.Printf("Projects for user %s (%d):\n", currentUser.FullName, currentUserID)
		fmt.Println("---------------------------------------------------------------------------------------------------------")
		fmt.Printf("%-60s %-15s %-15s %-15s\n", "Project Title", "Project Number", "Project Item", "Needs Ticket")
		fmt.Println("---------------------------------------------------------------------------------------------------------")
		for _, p := range projects {
			fmt.Printf("%-60.60s %-15d %-15d %-15t\n", p.ProjectTitle, p.ProjectNumber, p.EmployeeLineNumber, p.ProjectNeedTicket)
		}
		fmt.Println("---------------------------------------------------------------------------------------------------------")
	},
}
