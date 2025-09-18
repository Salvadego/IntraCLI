package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/Salvadego/IntraCLI/config"
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
	editFilterName    string
)

func init() {
	editCmd.Flags().IntVarP(&editTimesheetID, "id", "i", 0, "Timesheet ID to edit (skip if using filter)")
	editCmd.Flags().StringVarP(&editDescription, "description", "d", "", "New description")
	editCmd.Flags().StringVarP(&editHours, "hours", "H", "", "New hours")
	editCmd.Flags().StringVarP(&editTicket, "ticket", "t", "", "New ticket number")
	editCmd.Flags().StringVarP(&editProjectAlias, "project-alias", "p", "", "New project alias")
	editCmd.Flags().StringVarP(&editDate, "date", "D", "", "New date")
	editCmd.Flags().StringVarP(&editTimesheetType, "type", "T", "", "New timesheet type")
	editCmd.Flags().BoolVarP(&editUseEditor, "editor", "e", false, "Open editor for editing")
	editCmd.Flags().StringVar(&editFilterName, "filter", "", "Use a saved filter for batch editing")

	editCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)
	editCmd.RegisterFlagCompletionFunc("project-alias", projectAliasCompletionFunc)
	editCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)

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

		if editTimesheetID != 0 {
			// Fetch single timesheet
			all, err := client.Timesheet.GetTimesheets(ctx, currentUserID, time.Now().Year(), time.Now().Month())
			if err != nil {
				log.Fatalf("Failed to fetch timesheets: %v", err)
			}
			for _, ts := range all {
				if ts.TimesheetID == editTimesheetID {
					timesheets = append(timesheets, ts)
					break
				}
			}
		} else if editFilterName != "" {
			filter, ok := cfg.SavedFilters[editFilterName]
			if !ok {
				log.Fatalf("Saved filter '%s' not found", editFilterName)
			}

			all, err := client.Timesheet.GetTimesheets(ctx, currentUserID, time.Now().Year(), time.Now().Month())
			if err != nil {
				log.Fatalf("Failed to fetch timesheets: %v", err)
			}

			timesheets = utils.ApplyFilter(all, filter, profile)
		} else {
			log.Fatal("Must provide either --id or --filter")
		}

		if len(timesheets) == 0 {
			fmt.Println("No timesheets matched the criteria.")
			return
		}

		for _, ts := range timesheets {
			// Parse hours
			hours := ts.Quantity
			if editHours != "" {
				h, err := parseDurationString(editHours)
				if err != nil {
					log.Fatalf("Invalid hours format: %v", err)
				}
				hours = h
			}

			// Date
			date := ts.DateDoc[:10]
			if editDate != "" {
				if _, err := time.Parse("2006-01-02", editDate); err != nil {
					log.Fatalf("Invalid date format: %v", err)
				}
				date = editDate
			}

			// Description
			desc := ts.Description
			if editDescription != "" {
				desc = editDescription
			}

			// Ticket
			ticket := ts.TicketNo
			if editTicket != "" {
				ticket = editTicket
			}

			// Project
			salesOrder := int(ts.SalesOrder)
			salesOrderLine := int(ts.SalesOrderLine)
			if editProjectAlias != "" {
				info, ok := profile.ProjectAliases[editProjectAlias]
				if !ok {
					log.Fatalf("Unknown project alias '%s'", editProjectAlias)
				}
				if info.NeedsTicket && ticket == "" {
					log.Fatalf("Project '%s' requires a ticket", editProjectAlias)
				}
				salesOrder = info.SalesOrder
				salesOrderLine = info.SalesOrderLine
			}

			// Timesheet type
			tsType := ts.TimesheetType
			if editTimesheetType != "" {
				if key, ok := utils.TimesheetTypeLookup[editTimesheetType]; ok {
					tsType = key
				} else {
					log.Fatalf("Unknown timesheet type '%s'", editTimesheetType)
				}
			}

			// Delete old
			if err := client.Timesheet.DeleteTimesheet(ctx, ts.TimesheetID); err != nil {
				log.Printf("Failed to delete timesheet %d: %v", ts.TimesheetID, err)
				continue
			}

			// Re-create via appoint
			entry := TimesheetEntry{
				Date:           date,
				Description:    desc,
				TicketNo:       ticket,
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

func getCurrentProfile(cfg *config.Config) (config.Profile, error) {
	name := cfg.DefaultProfile
	if profileName != "" {
		name = profileName
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return config.Profile{}, fmt.Errorf("profile '%s' not found", name)
	}
	return p, nil
}
