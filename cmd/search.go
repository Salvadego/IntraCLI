package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/mantis/mantis"

	"github.com/spf13/cobra"
)

var employeeNameSearch string
var createProfileName string

const supervisorID int = 1031856

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

		employees, err := cache.ReadFromCache[mantis.S_Employee](cache.EmployeeListCacheFileName)
		if err != nil {
			employees, err = mantisClient.Employee.GetEmployeeList(mantisCtx, supervisorID)
			if err != nil {
				log.Fatalf("Error getting employees: %v", err)
			}
			err = cache.WriteToCache(cache.EmployeeListCacheFileName, employees)
			if err != nil {
				log.Fatalf("Failed to write to cache: %v", err)
			}
		}

		var S_Employee mantis.S_Employee
		for _, emp := range employees {
			if int(emp.AdUserID) == employee.UserID {
				S_Employee = emp
				break
			}
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
					cfg.Profiles[createProfileName] = mergeProfile(existing, employee, S_Employee)
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
				SUserID:        S_Employee.SUserID,
				LType:          S_Employee.TipoLider,
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
		fmt.Printf("  SUser ID: %s\n", S_Employee.SUserID)
		fmt.Printf("  Email: %s\n", employee.Email)
		fmt.Printf("  Daily Journey: %.2f\n", employee.DailyJourney)
	},
}

func mergeProfile(old config.Profile, emp mantis.Employee, sEmp mantis.S_Employee) config.Profile {
	p := old

	p.EmployeeName = emp.FullName
	p.Email = emp.Email
	p.EmployeeCode = emp.EmployeeCode
	p.UserID = emp.UserID
	p.DailyJourney = emp.DailyJourney
	p.SUserID = sEmp.SUserID
	p.LType = sEmp.TipoLider

	if p.ProjectAliases == nil {
		p.ProjectAliases = map[string]config.ProjectAlias{}
	}

	return p
}
