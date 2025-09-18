package cmd

import (
	"fmt"
	"log"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/spf13/cobra"
)

var (
	filterSaveName    string
	filterList        bool
	fromDate          string
	toDate            string
	filterTicket      string
	filterProject     string
	filterType        string
	filterDescription string
	filterQuantity    string
	hasTicketOnly     bool
)

func init() {
	filterTimesheetsCmd.Flags().StringVar(&filterSaveName, "save", "", "Save filter with given name")
	filterTimesheetsCmd.Flags().BoolVar(&filterList, "list", false, "List saved filters")
	filterTimesheetsCmd.Flags().StringVar(&fromDate, "from", "", "Filter from date (YYYY-MM-DD)")
	filterTimesheetsCmd.Flags().StringVar(&toDate, "to", "", "Filter to date (YYYY-MM-DD)")
	filterTimesheetsCmd.Flags().StringVar(&filterTicket, "ticket", "", "Filter by ticket number (contains)")
	filterTimesheetsCmd.Flags().StringVar(&filterProject, "project", "", "Filter by project (substring match)")
	filterTimesheetsCmd.Flags().BoolVar(&hasTicketOnly, "has-ticket-only", false, "Only timesheets with a ticket")
	filterTimesheetsCmd.Flags().StringVar(&filterType, "type", "", "Filter by timesheet type")
	filterTimesheetsCmd.Flags().StringVar(&filterDescription, "description", "", "Filter by description (substring match)")
	filterTimesheetsCmd.Flags().StringVar(&filterQuantity, "quantity", "", "Filter by quantity [>=|<=|=|<|>]<number>")

	filterTimesheetsCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)
	filterTimesheetsCmd.RegisterFlagCompletionFunc("project-alias", projectAliasCompletionFunc)
	filterTimesheetsCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)

	rootCmd.AddCommand(filterTimesheetsCmd)
}

var filterTimesheetsCmd = &cobra.Command{
	Use:   "filter-timesheets",
	Short: "Manage timesheet filters",
	Long: `Create, save, and list reusable filters for timesheets.
Examples:
  filter-timesheets --save myproj --project ProjectX --from 2025-07-01
  filter-timesheets --list`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.InitializeConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if filterList {
			if len(cfg.SavedFilters) == 0 {
				fmt.Println("No saved filters")
				return
			}
			fmt.Println("Saved filters:")
			for name, f := range cfg.SavedFilters {
				fmt.Printf(" - %s: %+v\n", name, f)
			}
			return
		}

		// Build filter from flags
		filter := types.TimesheetFilter{
			Name:          filterSaveName,
			FromDate:      fromDate,
			ToDate:        toDate,
			Ticket:        filterTicket,
			Project:       filterProject,
			HasTicketOnly: hasTicketOnly,
			Type:          filterType,
			Description:   filterDescription,
			Quantity:      filterQuantity,
		}

		if filterSaveName != "" {
			if cfg.SavedFilters == nil {
				cfg.SavedFilters = make(map[string]types.TimesheetFilter)
			}
			cfg.SavedFilters[filterSaveName] = filter
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatalf("Failed to save filter: %v", err)
			}
			fmt.Printf("Filter '%s' saved.\n", filterSaveName)
		} else {
			fmt.Println("No action taken. Use --save <name> to save or --list to list filters.")
		}
	},
}
