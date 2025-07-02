package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var timesheetID int

func init() {
	deleteTimesheetCmd.Flags().IntVarP(&timesheetID, "id", "i", 0, "ID of the timesheet to delete (required)")
	deleteTimesheetCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(deleteTimesheetCmd)
}

var deleteTimesheetCmd = &cobra.Command{
	Use:   "delete-timesheet",
	Short: "Delete a specific timesheet entry",
	Long:  `Deletes a timesheet entry from Mantis using its unique ID. Use 'list-timesheets' to find IDs.`,
	Run: func(cmd *cobra.Command, args []string) {
		if timesheetID == 0 {
			log.Fatal("Error: Timesheet ID cannot be 0. Please provide a valid ID using --id (-i).")
		}

		fmt.Printf("Attempting to delete timesheet with ID: %d\n", timesheetID)
		err := mantisClient.Timesheet.DeleteTimesheet(mantisCtx, timesheetID)
		if err != nil {
			log.Fatalf("Error deleting timesheet %d: %v", timesheetID, err)
		}

		fmt.Printf("Successfully deleted timesheet with ID: %d\n", timesheetID)
	},
}
