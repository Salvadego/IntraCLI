package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/spf13/cobra"
)

func init() {
	listTimesheetsCmd.Flags().IntVarP(&calYear, "year", "y", now.Year(), "Year to show")
	listTimesheetsCmd.Flags().StringVar(&filterName, "filter", "", "Apply a saved filter")
	listTimesheetsCmd.Flags().IntVarP(&calMonth, "month", "m", int(now.Month()), "Month to show (1-12)")

	listTimesheetsCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)
	rootCmd.AddCommand(listTimesheetsCmd)
}

var (
	meses = [...]string{
		"jan", "fev", "mar", "abr", "mai", "jun",
		"jul", "ago", "set", "out", "nov", "dez",
	}
)

var listTimesheetsCmd = &cobra.Command{
	Use:   "list-timesheets",
	Short: "List your timesheets for the current period",
	Long: `Retrieves and displays timesheet entries for the current configured
	period (based on Mantis's 26th of month logic) for the user in the default
	profile.`,
	Run: func(cmd *cobra.Command, args []string) {

		timesheets, err := mantisClient.Timesheet.GetTimesheets(
			mantisCtx,
			currentUserID,
			calYear,
			time.Month(calMonth))

		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}

		profile, profileExists := appConfig.Profiles[currentProfileName]
		if !profileExists {
			log.Fatalf("Profile '%s' not found", currentProfileName)
		}

		if filterName != "" {
			f, ok := appConfig.SavedFilters[filterName]
			if !ok {
				log.Fatalf("Filter '%s' not found", filterName)
			}
			timesheets = utils.ApplyFilter(timesheets, f, profile)
		}

		if err != nil {
			log.Fatalf("Error getting timesheets: %v", err)
		}

		filename := fmt.Sprintf(cache.TimesheetsCacheFileName, currentUserID)
		err = cache.WriteToCache(filename, timesheets)
		if err != nil {
			log.Printf("Warning: Failed to write timesheets to cache: %v", err)
		}

		if len(timesheets) == 0 {
			fmt.Println("No timesheets found for the current period.")
			return
		}

		fmt.Printf(
			"Timesheets for user %s (%d) for the current period:\n",
			currentUser.FullName,
			currentUserID,
		)

		printBar()
		fmt.Printf(
			"%-10s %-14s %-15s %-15s %-40.40s %-10s\n",
			"ID", "TimesheetType",
			"Date",
			"Hours",
			"Description",
			"Ticket",
		)
		printBar()

		for _, ts := range timesheets {
			parsedDate, err := time.Parse("2006-01-02T15:04:05Z", ts.DateDoc)

			if err != nil {
				continue
			}

			parsedTimesheetType, ok := types.TimesheetTypeInverseLookup[ts.TimesheetType]

			if !ok {
				continue
			}

			mes := meses[int(parsedDate.Month())-1]
			formatted := fmt.Sprintf("%d de %s %d", parsedDate.Day(), mes, parsedDate.Year())

			fmt.Printf(
				"%-10d %-14s %-15s %-15.2f %-40.40s %-10s\n",
				ts.TimesheetID,
				parsedTimesheetType,
				formatted,
				ts.Quantity,
				ts.Description,
				ts.TicketNo,
			)

		}
		printBar()
	},
}

func printBar() {
	fmt.Println(`--------------------------------------------------------------------------------------------------------------`)
}
