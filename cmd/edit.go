package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/config"
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

		if editUseEditor {
			processEditEditorFile(timesheets, profile, client, currentUserID, ctx)
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

func processEditEditorFile(timesheets []mantis.TimesheetsResponse, profile config.Profile, client *mantis.Client, userID int, ctx context.Context) {
	path := filepath.Join(os.Getenv("HOME"), ".local", "share", "intracli")
	os.MkdirAll(path, 0755)
	file := filepath.Join(path, "EDIT_TIMESHEETS")

	cache.WriteToCache("undo_timesheets.json", timesheets)

	var sb strings.Builder
	sb.WriteString("# IntraCLI Edit Mode\n")
	sb.WriteString("# Modifying these blocks will delete the old entry and create a new one.\n")
	sb.WriteString("# Format: description, hours, date, project-alias, ticket, type\n\n")

	for _, ts := range timesheets {
		currentAlias := ""
		for alias, info := range profile.ProjectAliases {
			if info.SalesOrder == int(ts.SalesOrder) && info.SalesOrderLine == int(ts.SalesOrderLine) {
				currentAlias = alias
				break
			}
		}

		fmt.Fprintf(&sb, "id: %d\n", ts.TimesheetID)
		fmt.Fprintf(&sb, "description: %s\n", ts.Description)
		fmt.Fprintf(&sb, "hours: %.2f\n", ts.Quantity)
		fmt.Fprintf(&sb, "date: %s\n", ts.DateDoc[:10])
		fmt.Fprintf(&sb, "project-alias: %s\n", currentAlias)
		fmt.Fprintf(&sb, "ticket: %s\n", ts.TicketNo)
		fmt.Fprintf(&sb, "type: %s\n", ts.TimesheetType)
		sb.WriteString("---\n")
	}

	if err := os.WriteFile(file, []byte(sb.String()), 0644); err != nil {
		log.Fatalf("Failed to write temporary file: %v", err)
	}
	initialContent, _ := os.ReadFile(file)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Using "sh -c" helps handle editors with flags like "code --wait"
	editorArgs := append(strings.Fields(editor), file)
	cmd := exec.Command(editorArgs[0], editorArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error opening editor: %v", err)
	}

	contentBytes, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read edited file: %v", err)
	}

	if string(initialContent) == string(contentBytes) {
		fmt.Println("No changes detected. Aborting edit.")
		return
	}

	blocks := strings.SplitSeq(string(contentBytes), "---")
	for block := range blocks {
		lines := strings.Split(block, "\n")
		entryMap := make(map[string]string)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				entryMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		idStr, ok := entryMap["id"]
		if !ok || idStr == "" {
			continue
		}
		oldID, _ := strconv.Atoi(idStr)

		alias := entryMap["project-alias"]
		projectInfo, ok := profile.ProjectAliases[alias]
		if !ok {
			log.Printf("Skipping ID %d: unknown project-alias '%s'", oldID, alias)
			continue
		}

		parsedHours, err := parseDurationString(entryMap["hours"])
		if err != nil {
			log.Printf("Skipping ID %d: invalid hours: %v", oldID, err)
			continue
		}

		tsTypeKey := "N"
		if key, ok := types.TimesheetTypeLookup[entryMap["type"]]; ok {
			tsTypeKey = key
		} else {
			if len(entryMap["type"]) == 1 {
				tsTypeKey = entryMap["type"]
			}
		}

		entry := TimesheetEntry{
			Date:           entryMap["date"],
			Description:    entryMap["description"],
			TicketNo:       entryMap["ticket"],
			TimesheetType:  tsTypeKey,
			Hours:          parsedHours,
			SalesOrder:     projectInfo.SalesOrder,
			SalesOrderLine: projectInfo.SalesOrderLine,
		}

		fmt.Printf("Updating timesheet %d...\n", oldID)
		fmt.Println(entry)
		// if err := client.Timesheet.DeleteTimesheet(ctx, oldID); err != nil {
		// 	log.Printf("Failed to delete old timesheet %d: %v", oldID, err)
		// 	continue
		// }

		// appoint(client, userID, entry, ctx)
	}
}
