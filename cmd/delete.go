package cmd

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var timesheetID int

func init() {
	deleteTimesheetCmd.Flags().IntVarP(&timesheetID, "id", "i", 0, "ID of the timesheet to delete (required)")
	deleteTimesheetCmd.MarkFlagRequired("id")

	deleteTimesheetCmd.RegisterFlagCompletionFunc("id",
		func(
			cmd *cobra.Command,
			args []string,
			toComplete string,
		) ([]string, cobra.ShellCompDirective) {
			filename := fmt.Sprintf(cache.TimesheetsCacheFileName, currentUserID)
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
		},
	)

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
