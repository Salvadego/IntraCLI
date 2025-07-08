package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/mantis/mantis"

	"github.com/spf13/cobra"
)

var (
	ticket        string
	description   string
	hoursString   string
	date          string
	projectAlias  string
	profileName   string
	useEditor     bool
	timesheetType string
)

func init() {
	flagSet := appointCmd.Flags()
	flagSet.StringVarP(&description, "description", "d", "", "Description of the appointment (required)")
	flagSet.StringVarP(&hoursString, "hours", "H", "", "Hours spent on the appointment (required)")
	flagSet.StringVarP(&date, "date", "D", time.Now().Format("2006-01-02"), "Date of the appointment (YYYY-MM-DD, defaults to today)")
	flagSet.StringVarP(&projectAlias, "project-alias", "p", "", "Alias for the project (required)")
	flagSet.StringVarP(&ticket, "ticket", "t", "", "Optional ticket number associated with the appointment")
	flagSet.StringVarP(&timesheetType, "type", "T", "", "Timesheet Type to make the appointment")
	flagSet.BoolVarP(&useEditor, "editor", "e", false, "Open default editor to create appointments")

	appointCmd.RegisterFlagCompletionFunc("type",
		func(
			cmd *cobra.Command,
			args []string,
			toComplete string,
		) ([]string, cobra.ShellCompDirective) {

			var timesheetTypes []string
			for t := range timesheetTypeLookup {
				if strings.HasPrefix(t, toComplete) {
					timesheetTypes = append(timesheetTypes, t)
				}
			}

			return timesheetTypes, cobra.ShellCompDirectiveNoFileComp
		})

	appointCmd.RegisterFlagCompletionFunc("project-alias",
		func(
			cmd *cobra.Command,
			args []string,
			toComplete string,
		) ([]string, cobra.ShellCompDirective) {
			cfg, err := config.InitializeConfig()
			if err != nil {
				log.Printf("Error loading config for completion: %v", err)
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			currentProfileName := cfg.DefaultProfile
			if profileName != "" {
				currentProfileName = profileName
			}

			profile, ok := cfg.Profiles[currentProfileName]
			if !ok {
				log.Printf(
					"Default profile '%s' not found for completion.",
					cfg.DefaultProfile,
				)
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var aliases []string
			for alias := range profile.ProjectAliases {
				if strings.HasPrefix(alias, toComplete) {
					aliases = append(aliases, alias)
				}
			}
			return aliases, cobra.ShellCompDirectiveNoFileComp
		})

	rootCmd.AddCommand(appointCmd)
}

var appointCmd = &cobra.Command{
	Use:   "appoint",
	Short: "Make a new appointment",
	Long: `This command allows you to create a new timesheet appointment with
	details like description, hours, date, ticket number (optional), and a
	project alias.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := mantisClient
		ctx := mantisCtx
		cfg := appConfig
		userID := currentUserID

		currentProfileName := cfg.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}

		profile, ok := cfg.Profiles[currentProfileName]
		if !ok {
			log.Fatalf("Profile '%s' not found in configuration. Please check your config.yaml.", currentProfileName)
		}

		if useEditor {
			processEditorFile(profile, client, userID, ctx)
			return
		}

		if description == "" {
			log.Fatalf("Missing required flags: --description")
		}

		if hoursString == "" {
			log.Fatalf("Missing required flags: --hours")
		}

		if projectAlias == "" {
			log.Fatalf("Missing required flags: --project-alias")
		}

		projectInfo, ok := profile.ProjectAliases[projectAlias]
		if !ok {
			log.Fatalf("Project alias '%s' not found in your default profile.", projectAlias)
		}

		if projectInfo.NeedsTicket && ticket == "" {
			log.Fatalf("Error: Project '%s' requires a ticket number. Please provide one using --ticket (-t).", projectAlias)
		}

		parsedHours, err := parseDurationString(hoursString)
		if err != nil {
			log.Fatalf("Invalid hours format: %v", err)
		}

		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatalf("Invalid date format. Please use YYYY-MM-DD. Error: %v", err)
		}

		timesheetTypeKey := "N"
		if key, ok := timesheetTypeLookup[timesheetType]; ok {
			timesheetTypeKey = key
		}

		timesheetEntry := TimesheetEntry{
			Date:           parsedDate.Format("2006-01-02"),
			Description:    description,
			TicketNo:       ticket,
			TimesheetType:  timesheetTypeKey,
			Hours:          parsedHours,
			SalesOrder:     projectInfo.SalesOrder,
			SalesOrderLine: projectInfo.SalesOrderLine,
		}

		fmt.Printf("Attempting to create appointment:\n%+v\n", timesheetEntry)

		appoint(client, userID, timesheetEntry, ctx)
	},
}

func parseDurationString(durationStr string) (float64, error) {
	totalHours := 0.0
	re := regexp.MustCompile(`(\d+)([wdhm])`)
	matches := re.FindAllStringSubmatch(durationStr, -1)

	if len(matches) == 0 && durationStr != "" {
		hours, err := strconv.ParseFloat(durationStr, 64)
		if err == nil {
			return hours, nil
		}
		return 0, fmt.Errorf(
			"invalid duration format: %s. Expected format like '8h' or '1d 2h'",
			durationStr)
	}

	for _, match := range matches {
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration string: %s", match[1])
		}
		unit := match[2]

		switch unit {
		case "d":
			totalHours += value * 8 // 1 day = 8 hours
		case "h":
			totalHours += value // 1 hour
		case "m":
			totalHours += value / 60 // 1 minute = 1/60 hours
		default:
			return 0, fmt.Errorf("unknown unit '%s' in duration string", unit)
		}
	}

	return totalHours, nil
}

type TimesheetEntry struct {
	Date           string
	Description    string
	TicketNo       string
	TimesheetType  string
	Hours          float64
	SalesOrderLine int
	SalesOrder     int
}

func appoint(client *mantis.Client, userID int, entry TimesheetEntry, ctx context.Context) {
	timesheet := mantis.Timesheet{
		Fase:                nil,
		UserID:              userID,
		SalesOrderLine:      entry.SalesOrderLine,
		Quantity:            entry.Hours,
		SalesOrder:          entry.SalesOrder,
		Description:         entry.Description,
		DateDoc:             entry.Date + "T00:00:00Z",
		TicketNo:            entry.TicketNo,
		TicketContractTitle: "",
		TimesheetType:       entry.TimesheetType,
		TicketDescription:   "",
	}
	fmt.Println("Sending timesheet")

	errors, err := client.Timesheet.Create(ctx, timesheet)
	if len(errors.Errors) > 0 {
		fmt.Printf(
			"Error creating timesheet for %s: %s\n",
			entry.Date,
			errors.Errors[0].Message,
		)
		return
	}

	if err == nil {
		fmt.Printf(
			"Successfully created timesheet for %s with %.2f hours\n",
			entry.Date,
			entry.Hours,
		)
		return
	}

	fmt.Printf(
		"Error creating timesheet for %s: %s\n",
		entry.Date,
		err.Error(),
	)

}

func processEditorFile(profile config.Profile, client *mantis.Client, userID int, ctx context.Context) {
	path := filepath.Join(os.Getenv("HOME"), ".local", "share", "intracli")
	os.MkdirAll(path, 0755)
	file := filepath.Join(path, "EDIT_APPOINTMENTS")

	template := `# IntraCLI Appointment Editor
# Each block below represents an appointment.
# Format:
#
# description: Work on feature X
# hours: 2h
# date: 2025-07-08
# project-alias: projx
# ticket: 12345
# type: Normal
#
# Blank line separates appointments. Lines starting with '#' are ignored.
#
# Save and close the file to create appointments.

`

	err := os.WriteFile(file, []byte(template), 0644)
	if err != nil {
		log.Fatalf("Failed to write template to file: %v", err)
	}

	exec_editor := os.Getenv("EDITOR")
	if exec_editor == "" {
		exec_editor = "vim"
	}

	cmd := exec.Command(exec_editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error opening editor: %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read edited file: %v", err)
	}

	blocks := strings.SplitSeq(string(content), "\n\n")
	for block := range blocks {
		lines := strings.Split(block, "\n")
		entryMap := make(map[string]string)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			entryMap[key] = value
		}

		if entryMap["description"] == "" ||
			entryMap["hours"] == "" ||
			entryMap["project-alias"] == "" {
			continue
		}

		projectInfo, ok := profile.ProjectAliases[entryMap["project-alias"]]
		if !ok {
			log.Printf("Skipping block: unknown project-alias '%s'", entryMap["project-alias"])
			continue
		}

		if projectInfo.NeedsTicket && entryMap["ticket"] == "" {
			log.Printf("Skipping block: project '%s' requires ticket", entryMap["project-alias"])
			continue
		}

		parsedHours, err := parseDurationString(entryMap["hours"])
		if err != nil {
			log.Printf("Skipping block due to invalid hours: %v", err)
			continue
		}

		entryDate := time.Now().Format("2006-01-02")
		if entryMap["date"] != "" {
			_, err := time.Parse("2006-01-02", entryMap["date"])
			if err != nil {
				log.Printf("Invalid date, using today instead: %v", err)
			} else {
				entryDate = entryMap["date"]
			}
		}

		timesheetTypeKey := "N"
		if key, ok := timesheetTypeLookup[entryMap["type"]]; ok {
			timesheetTypeKey = key
		}

		timesheetEntry := TimesheetEntry{
			Date:           entryDate,
			Description:    entryMap["description"],
			TicketNo:       entryMap["ticket"],
			TimesheetType:  timesheetTypeKey,
			Hours:          parsedHours,
			SalesOrder:     projectInfo.SalesOrder,
			SalesOrderLine: projectInfo.SalesOrderLine,
		}

		fmt.Printf("Creating appointment: %+v\n", timesheetEntry)
		appoint(client, userID, timesheetEntry, ctx)
	}
}
