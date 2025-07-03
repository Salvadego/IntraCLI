package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/Salvadego/IntraCLI/config"

	"github.com/spf13/cobra"
)

var employeeNameSearch string
var createProfileName string

func init() {
	searchEmployeeCmd.Flags().StringVarP(&employeeNameSearch, "name", "n", "", "Name or part of the employee's name to search for")
	searchEmployeeCmd.Flags().StringVarP(&createProfileName, "create-profile", "c", "", "Create a new profile using the found employee (provide a profile name)")

	rootCmd.AddCommand(searchEmployeeCmd)
}

var searchEmployeeCmd = &cobra.Command{
	Use:   "search-employee",
	Short: "Search for an employee",
	Long:  `Searches for employees in Mantis by a given name or it's user ID. Useful for finding the exact 'employeeName' for your configuration profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		if employeeNameSearch == "" {
			log.Fatal("Error: Employee name to search for is required. Use --name (-n).")
		}

		employee, err := mantisClient.Employee.GetEmployeeByName(mantisCtx, employeeNameSearch)
		if err != nil {
			fmt.Printf("No exact match found for '%s'. Attempting partial match or broader search logic...\n", employeeNameSearch)

			if strings.Contains(err.Error(), "não foi possível encontrar funcionário ativo") {
				fmt.Printf("No active employee found matching '%s'. Please try a different name or part of the name.\n", employeeNameSearch)
			} else {
				log.Fatalf("Error searching for employee '%s': %v", employeeNameSearch, err)
			}

			return
		}

		if createProfileName != "" {
			cfg, err := config.LoadConfig()
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}

			if _, exists := cfg.Profiles[createProfileName]; exists {
				log.Fatalf("Profile '%s' already exists. Choose a different name.", createProfileName)
			}

			newProfile := config.Profile{
				EmployeeName:   employee.FullName,
				UserID:         employee.UserID,
				ProjectAliases: map[string]config.ProjectAlias{},
			}

			cfg.Profiles[createProfileName] = newProfile

			err = config.SaveConfig(cfg)
			if err != nil {
				log.Fatalf("Error saving config: %v", err)
			}

			fmt.Printf("Profile '%s' created successfully.\n", createProfileName)
		}

		fmt.Printf("Found employee:\n")
		fmt.Printf("  Full Name: %s\n", employee.FullName)
		fmt.Printf("  Employee Code: %d\n", employee.EmployeeCode)
		fmt.Printf("  User ID: %d\n", employee.UserID)
		fmt.Printf("  Email: %s\n", employee.Email)
	},
}
