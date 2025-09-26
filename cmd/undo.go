package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(undoCmd)
}

var undoCmd = &cobra.Command{
	Use:   "undo-timesheet",
	Short: "Restore the last deleted/recreated timesheets",
	Run: func(cmd *cobra.Command, args []string) {
		toDeleteCacheFile := "to_delete_timesheets.json"
		var cached []mantis.TimesheetsResponse

		cached, err := cache.ReadFromCache[mantis.TimesheetsResponse](toDeleteCacheFile)
		if err != nil {
			log.Fatalf("Failed to read to delete cache: %v", err)
		}
		if len(cached) == 0 {
			fmt.Println("No to delete information available.")
			return
		}

		client := mantisClient
		ctx := mantisCtx
		for _, ts := range cached {
			client.Timesheet.DeleteTimesheet(ctx, ts.TimesheetID)
		}

		fmt.Println("Deletion completed.")
		toDeleteCachePath, _ := cache.GetCacheFilePath(toDeleteCacheFile)
		_ = os.Remove(toDeleteCachePath)

		undoCacheFile := "undo_timesheets.json"

		cached, err = cache.ReadFromCache[mantis.TimesheetsResponse](undoCacheFile)
		if err != nil {
			log.Fatalf("Failed to read undo cache: %v", err)
		}
		if len(cached) == 0 {
			fmt.Println("No undo information available.")
			return
		}

		for _, ts := range cached {
			entry := TimesheetEntry{
				Date:           ts.DateDoc[:10],
				Description:    ts.Description,
				TicketNo:       ts.TicketNo,
				TimesheetType:  ts.TimesheetType,
				Hours:          ts.Quantity,
				SalesOrder:     int(ts.SalesOrder),
				SalesOrderLine: int(ts.SalesOrderLine),
			}
			fmt.Printf("Restoring timesheet: %+v\n", entry)
			appoint(client, currentUserID, entry, ctx)
		}

		fmt.Println("Undo completed.")

		fmt.Printf("Removing undo cache file: %s\n", undoCacheFile)

		cachePath, _ := cache.GetCacheFilePath(undoCacheFile)
		_ = os.Remove(cachePath)

	},
}
