package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var now = time.Now()

var listFilterFlag string

func init() {
	listTimesheetsCmd.Flags().IntVarP(&calYear, "year", "y", now.Year(), "Year to show")
	listTimesheetsCmd.Flags().IntVarP(&calMonth, "month", "m", int(now.Month()), "Month to show (1-12)")
	listTimesheetsCmd.Flags().StringVar(&listFilterFlag, "filter", "", "Filter: raw qlvm query or @savedName")

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
	Run: func(cmd *cobra.Command, args []string) {
		timesheets, err := mantisClient.Timesheet.GetTimesheets(
			mantisCtx, currentUserID, calYear, time.Month(calMonth),
		)
		if err != nil {
			log.Fatalf("Error getting timesheets: %v", err)
		}

		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}
		profile, profileExists := appConfig.Profiles[currentProfileName]
		if !profileExists {
			log.Fatalf("Profile '%s' not found", currentProfileName)
		}

		if listFilterFlag != "" {
			query := resolveFilter(listFilterFlag, appConfig.SavedFilters)
			timesheets, err = utils.Apply(query, timesheets, profile)
			if err != nil {
				log.Fatalf("Filter error: %v", err)
			}
		}

		filename := fmt.Sprintf(cache.TimesheetsCacheFileName, currentUserID, calYear, time.Month(calMonth))
		if err := cache.WriteToCache(filename, timesheets); err != nil {
			log.Printf("Warning: failed to write timesheets to cache: %v", err)
		}

		if len(timesheets) == 0 {
			fmt.Println("No timesheets found for the current period.")
			return
		}

		fmt.Printf(
			"Timesheets for user %s (%d) for the current period:\n\n",
			currentUser.FullName, currentUserID,
		)

		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal},
					Alignment:    tw.CellAlignment{Global: tw.AlignLeft},
					ColMaxWidths: tw.CellWidth{Global: 40},
				},
			}),
		)

		table.Header("ID", "TimesheetType", "Date", "Hours", "Description", "Ticket", "SalesOrder", "SalesOrderLine")

		for _, ts := range timesheets {
			parsedDate, err := time.Parse("2006-01-02T15:04:05Z", ts.DateDoc)
			if err != nil {
				continue
			}
			parsedType, ok := types.TimesheetTypeInverseLookup[ts.TimesheetType]
			if !ok {
				continue
			}
			mes := meses[int(parsedDate.Month())-1]
			formatted := fmt.Sprintf("%d de %s %d", parsedDate.Day(), mes, parsedDate.Year())

			table.Append(
				fmt.Sprintf("%d", ts.TimesheetID),
				parsedType,
				formatted,
				fmt.Sprintf("%.2f", ts.Quantity),
				ts.Description,
				ts.TicketNo,
				ts.SalesOrder,
				ts.SalesOrderLine,
			)
		}

		table.Render()
	},
}
