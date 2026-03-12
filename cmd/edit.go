package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

var (
	editTimesheetID   int
	editDescription   string
	editHours         string
	editTicket        string
	editProjectAlias  string
	editDate          string
	editTimesheetType string
	editUseEditor     bool
	editFilterFlag    string
)

func init() {
	editCmd.Flags().IntVarP(&editTimesheetID, "id", "i", 0, "Timesheet ID to edit")
	editCmd.Flags().StringVarP(&editDescription, "description", "d", "", "New description")
	editCmd.Flags().StringVarP(&editHours, "hours", "H", "", "New hours")
	editCmd.Flags().StringVarP(&editTicket, "ticket", "t", "", "New ticket number")
	editCmd.Flags().StringVarP(&editProjectAlias, "project-alias", "p", "", "New project alias")
	editCmd.Flags().StringVarP(&editDate, "date", "D", "", "New date")
	editCmd.Flags().StringVarP(&editTimesheetType, "type", "T", "", "New timesheet type")
	editCmd.Flags().BoolVarP(&editUseEditor, "editor", "e", false, "Open editor for editing")
	editCmd.Flags().StringVar(&editFilterFlag, "filter", "",
		"Batch-edit by filter: raw qlvm query or @savedName")

	editCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)
	editCmd.RegisterFlagCompletionFunc("project-alias", projectAliasCompletionFunc)
	editCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)
	editCmd.RegisterFlagCompletionFunc("id", timesheetIdCompletionFunc)

	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit-timesheet",
	Short: "Edit one or more timesheets by deleting and re-creating them",
	Run: func(cmd *cobra.Command, args []string) {
		client := mantisClient
		ctx := mantisCtx
		cfg := appConfig

		profile, err := getCurrentProfile(cfg)
		if err != nil {
			log.Fatal(err)
		}

		var timesheets []mantis.TimesheetsResponse

		switch {
		case editTimesheetID != 0:
			ts, err := client.Timesheet.Get(ctx, editTimesheetID)
			if err != nil {
				log.Fatalf("Failed to fetch timesheet %d: %v", editTimesheetID, err)
			}
			timesheets = append(timesheets, ts[0])

		case editFilterFlag != "":
			query := resolveFilter(editFilterFlag, appConfig.SavedFilters)

			all, err := client.Timesheet.GetTimesheets(ctx, currentUserID, time.Now().Year(), time.Now().Month())
			if err != nil {
				log.Fatalf("Failed to fetch timesheets: %v", err)
			}

			timesheets = utils.ApplyFilter(all, query, profile)

		default:
			log.Fatal("Must provide either --id or --filter")
		}

		if len(timesheets) == 0 {
			fmt.Println("No timesheets matched the criteria.")
			return
		}

		for _, ts := range timesheets {
			hours := ts.Quantity
			if editHours != "" {
				h, err := parseDurationString(editHours)
				if err != nil {
					log.Fatalf("Invalid hours format: %v", err)
				}
				hours = h
			}

			date := ts.DateDoc[:10]
			if editDate != "" {
				if _, err := time.Parse("2006-01-02", editDate); err != nil {
					log.Fatalf("Invalid date format: %v", err)
				}
				date = editDate
			}

			desc := ts.Description
			if editDescription != "" {
				desc = editDescription
			}

			tkn := ts.TicketNo
			if editTicket != "" {
				tkn = editTicket
			}

			salesOrder := int(ts.SalesOrder)
			salesOrderLine := int(ts.SalesOrderLine)
			if editProjectAlias != "" {
				info, ok := profile.ProjectAliases[editProjectAlias]
				if !ok {
					log.Fatalf("Unknown project alias '%s'", editProjectAlias)
				}
				if info.NeedsTicket && tkn == "" {
					log.Fatalf("Project '%s' requires a ticket", editProjectAlias)
				}
				salesOrder = info.SalesOrder
				salesOrderLine = info.SalesOrderLine
			}

			tsType := ts.TimesheetType
			if editTimesheetType != "" {
				key, ok := types.TimesheetTypeLookup[editTimesheetType]
				if !ok {
					log.Fatalf("Unknown timesheet type '%s'", editTimesheetType)
				}
				tsType = key
			}

			if err := client.Timesheet.DeleteTimesheet(ctx, ts.TimesheetID); err != nil {
				log.Printf("Failed to delete timesheet %d: %v", ts.TimesheetID, err)
				continue
			}

			entry := TimesheetEntry{
				Date:           date,
				Description:    desc,
				TicketNo:       tkn,
				TimesheetType:  tsType,
				Hours:          hours,
				SalesOrder:     salesOrder,
				SalesOrderLine: salesOrderLine,
			}

			fmt.Printf("Recreating timesheet: %+v\n", entry)
			appoint(client, currentUserID, entry, ctx)
		}
	},
}
