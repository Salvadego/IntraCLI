package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var (
	timesheetID      int
	deleteFilterFlag string
)

func init() {
	deleteTimesheetCmd.Flags().IntVarP(&timesheetID, "id", "i", 0, "ID of the timesheet to delete")
	deleteTimesheetCmd.Flags().StringVar(&deleteFilterFlag, "filter", "",
		"Batch-delete by filter: raw qlvm query or @savedName")

	deleteTimesheetCmd.RegisterFlagCompletionFunc("id", timesheetIdCompletionFunc)
	deleteTimesheetCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)

	rootCmd.AddCommand(deleteTimesheetCmd)
}

var deleteTimesheetCmd = &cobra.Command{
	Use:   "delete-timesheet",
	Short: "Delete a specific timesheet entry",
	Long:  `Deletes a timesheet entry by ID or batch-deletes by filter. Use 'list-timesheets' to find IDs.`,
	Run: func(cmd *cobra.Command, args []string) {
		if timesheetID == 0 && deleteFilterFlag == "" {
			log.Fatal("Error: no timesheet ID or filter provided. Use --id (-i) or --filter.")
		}

		client := mantisClient
		ctx := mantisCtx
		cfg := appConfig

		profile, err := getCurrentProfile(cfg)
		if err != nil {
			log.Fatal(err)
		}

		var timesheets []mantis.TimesheetsResponse

		if deleteFilterFlag != "" {
			query := resolveFilter(deleteFilterFlag, appConfig.SavedFilters)

			all, err := client.Timesheet.GetTimesheets(ctx, currentUserID, time.Now().Year(), time.Now().Month())
			if err != nil {
				log.Fatalf("Failed to fetch timesheets: %v", err)
			}

			timesheets = utils.ApplyFilter(all, query, profile)
		}

		if timesheetID != 0 {
			fmt.Printf("Attempting to delete timesheet: %d\n", timesheetID)
			if err := client.Timesheet.DeleteTimesheet(ctx, timesheetID); err != nil {
				log.Fatalf("Error deleting timesheet %d: %v", timesheetID, err)
			}
		}

		for _, ts := range timesheets {
			fmt.Printf("Attempting to delete timesheet: %d\n", ts.TimesheetID)
			if err := client.Timesheet.DeleteTimesheet(ctx, ts.TimesheetID); err != nil {
				log.Fatalf("Error deleting timesheet %d: %v", ts.TimesheetID, err)
			}
		}
	},
}
