package cmd

import (
	"fmt"
	"log"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/spf13/cobra"
)

var (
	filterDaysFromDate      string
	filterDaysToDate        string
	filterDaysMinHours      float64
	filterDaysNegate        bool
	saveDaysFilter          string
	filterDaysDelete        string
	filterDaysList          bool
	filterDaysProject       string
	filterDaysUser          string
	filterDaysStatus        string
	filterDaysHasTicketOnly bool
)

func init() {
	filterDaysCmd.Flags().BoolVar(&filterDaysList, "list", false, "List saved filters")
	filterDaysCmd.Flags().StringVar(&saveDaysFilter, "save", "", "Save filter with given name")
	filterDaysCmd.Flags().StringVar(&filterDaysDelete, "delete", "", "Delete a filter")
	filterDaysCmd.Flags().StringVar(&filterDaysFromDate, "from", "", "Start date (YYYY-MM-DD)")
	filterDaysCmd.Flags().StringVar(&filterDaysToDate, "to", "", "End date (YYYY-MM-DD)")
	filterDaysCmd.Flags().Float64Var(&filterDaysMinHours, "min-hours", 0, "Minimum daily hours to check")
	filterDaysCmd.Flags().BoolVar(&filterDaysNegate, "negate", false, "Negate the filter (show days meeting the criteria)")
	filterDaysCmd.Flags().BoolVar(&filterDaysHasTicketOnly, "has-ticket-only", false, "Filter by tickets only")
	filterDaysCmd.Flags().StringVar(&filterDaysProject, "project", "", "Filter by project name")
	filterDaysCmd.Flags().StringVar(&filterDaysUser, "user", "", "Filter by user name")
	filterDaysCmd.Flags().StringVar(&filterDaysStatus, "status", "", "Filter by status (OK, MISSING, OVERTIME)")

	filterDaysCmd.RegisterFlagCompletionFunc("delete", filterDaysNameCompletionFunc)

	rootCmd.AddCommand(filterDaysCmd)
}

var filterDaysCmd = &cobra.Command{
	Use:   "filter-days",
	Short: "Manage filter days",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.InitializeConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if filterDaysList {
			if len(cfg.SavedDayFilters) == 0 {
				fmt.Println("No saved filters")
				return
			}
			fmt.Println("Saved filters:")
			for name, f := range cfg.SavedDayFilters {
				fmt.Printf(" - %s: %+v\n", name, f)
			}
			return
		}

		if filterDaysDelete != "" {
			delete(cfg.SavedDayFilters, filterDaysDelete)
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatalf("Failed to save filter: %v", err)
			}
			fmt.Printf("Filter '%s' deleted.\n", filterDaysDelete)
			return
		}

		filter := types.DailyFilter{
			FromDate:      filterDaysFromDate,
			ToDate:        filterDaysToDate,
			MinDailyHours: filterDaysMinHours,
			Negate:        filterDaysNegate,
			HasTicketOnly: filterDaysHasTicketOnly,
			Project:       filterDaysProject,
			User:          filterDaysUser,
			Status:        filterDaysStatus,
		}

		if cfg.SavedDayFilters == nil {
			cfg.SavedDayFilters = make(map[string]types.DailyFilter)
		}

		if saveDaysFilter != "" {
			cfg.SavedDayFilters[saveDaysFilter] = filter
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatalf("Failed to save filter: %v", err)
			}
			fmt.Printf("Filter '%s' saved.\n", saveDaysFilter)
		}

	},
}
