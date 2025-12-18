package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/mantis/mantis"

	"github.com/spf13/cobra"
)

const allEmployeesRoleID = 1000333

var employeeNameSearch string
var createProfileName string

func init() {
	searchEmployeeCmd.Flags().StringVarP(&employeeNameSearch, "name", "n", "", "Name or part of the employee's name to search for")
	searchEmployeeCmd.Flags().StringVarP(&createProfileName, "create-profile", "c", "", "Create or update a profile using the found employee (provide a profile name)")

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
			cfg, err := config.InitializeConfig()
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}

			if existing, exists := cfg.Profiles[createProfileName]; exists {

				log.Printf("Profile '%s' already exists. Choose a different name.\n", createProfileName)
				log.Println("Do you wish to update it? ")
				fmt.Print("y/n: ")
				var choice string
				_, _ = fmt.Scanln(&choice)

				if choice == "y" {
					log.Println("Updating profile...")
					cfg.Profiles[createProfileName] = mergeProfile(existing, employee)
					err = config.SaveConfig(cfg)
					if err != nil {
						log.Fatalf("Error saving config: %v", err)
					}
					fmt.Printf("Profile '%s' updated successfully.\n", createProfileName)
					return
				} else {
					log.Println("Aborting...")
					return
				}
			}

			newProfile := config.Profile{
				EmployeeName:   employee.FullName,
				DailyJourney:   employee.DailyJourney,
				Email:          employee.Email,
				EmployeeCode:   employee.EmployeeCode,
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
		fmt.Printf("  Daily Journey: %.2f\n", employee.DailyJourney)
	},
}

func mergeProfile(old config.Profile, emp mantis.Employee) config.Profile {
	p := old

	p.EmployeeName = emp.FullName
	p.Email = emp.Email
	p.EmployeeCode = emp.EmployeeCode
	p.UserID = emp.UserID
	p.DailyJourney = emp.DailyJourney

	if p.ProjectAliases == nil {
		p.ProjectAliases = map[string]config.ProjectAlias{}
	}

	return p
}
