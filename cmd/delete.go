package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var timesheetID int

func init() {
	deleteTimesheetCmd.Flags().IntVarP(&timesheetID, "id", "i", 0, "ID of the timesheet to delete (required)")
	deleteTimesheetCmd.Flags().StringVar(&filterName, "filter", "", "Filter to use for deleting timesheets")

	deleteTimesheetCmd.RegisterFlagCompletionFunc("id", timesheetIdCompletionFunc)
	deleteTimesheetCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)

	rootCmd.AddCommand(deleteTimesheetCmd)
}

var deleteTimesheetCmd = &cobra.Command{
	Use:   "delete-timesheet",
	Short: "Delete a specific timesheet entry",
	Long:  `Deletes a timesheet entry from Mantis using its unique ID. Use 'list-timesheets' to find IDs.`,
	Run: func(cmd *cobra.Command, args []string) {
		if timesheetID != 0 && filterName == "" {
			log.Fatal("Error: No timesheet ID or filter provided. Please provide a valid ID using --id (-i) or --filter.")
		}

		var timesheets []mantis.TimesheetsResponse
		client := mantisClient
		ctx := mantisCtx
		cfg := appConfig

		profile, err := getCurrentProfile(cfg)
		if err != nil {
			log.Fatal(err)
		}
		if filterName != "" {
			filter, ok := cfg.SavedFilters[filterName]
			if !ok {
				log.Fatalf("Saved filter '%s' not found", filterName)
			}

			all, err := client.Timesheet.GetTimesheets(ctx, currentUserID, time.Now().Year(), time.Now().Month())
			if err != nil {
				log.Fatalf("Failed to fetch timesheets: %v", err)
			}

			timesheets = utils.ApplyFilter(all, filter, profile)
		}

		if timesheetID != 0 {
			fmt.Printf("Attempting to delete timesheet: %d\n", timesheetID)
			err = mantisClient.Timesheet.DeleteTimesheet(mantisCtx, timesheetID)
			if err != nil {
				log.Fatalf("Error deleting timesheet %d: %v", timesheetID, err)
			}
		}

		for _, ts := range timesheets {
			fmt.Printf("Attempting to delete timesheet: %d\n", ts.TimesheetID)
			err = mantisClient.Timesheet.DeleteTimesheet(mantisCtx, ts.TimesheetID)
			if err != nil {
				log.Fatalf("Error deleting timesheet %d: %v", timesheetID, err)
			}
		}

	},
}
