package cmd

import (
	"fmt"
	"log"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/spf13/cobra"
)

var (
	saveDaysFilter   string
	filterDaysDelete string
	filterDaysList   bool
)

func init() {
	filterDaysCmd.Flags().BoolVar(&filterDaysList, "list", false, "List saved day filters")
	filterDaysCmd.Flags().StringVar(&saveDaysFilter, "save", "", "Save the query with the given name")
	filterDaysCmd.Flags().StringVar(&filterDaysDelete, "delete", "", "Delete the named filter")

	filterDaysCmd.RegisterFlagCompletionFunc("delete", filterDaysNameCompletionFunc)

	rootCmd.AddCommand(filterDaysCmd)
}

// filterDaysCmd manages saved qlvm query strings for daily-summary filtering.
//
// Saving a filter:
//
//	intracli filter-days --save missing "status = MISSING"
//
// Using a saved filter:
//
//	intracli date-summary --filter-day missing
//
// Query language quick-reference (qlvm) — fields available:
//
//	date     YYYY-MM-DD  e.g.  date >= 2025-07-01
//	hours    float64     e.g.  hours < 4
//	status   string      e.g.  status = MISSING   (OK | MISSING | OVERTIME)
//	project  string      e.g.  project = myalias
//	user     string      e.g.  user = "John Doe"
//	week     int         e.g.  week = 27
//	month    string      e.g.  month = July
var filterDaysCmd = &cobra.Command{
	Use:   "filter-days [query]",
	Short: "Manage saved daily-summary filters (qlvm query strings)",
	Long: `Save, list, or delete named qlvm query strings used to filter daily summaries.

The optional positional argument is the raw qlvm query string to save.

Examples:
  intracli filter-days --save missing  "status = MISSING"
  intracli filter-days --save overtime "status = OVERTIME"
  intracli filter-days --save q3       "date >= 2025-07-01 AND date <= 2025-09-30"
  intracli filter-days --list
  intracli filter-days --delete missing`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.InitializeConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if filterDaysList {
			if len(cfg.SavedDayFilters) == 0 {
				fmt.Println("No saved day filters.")
				return
			}
			fmt.Println("Saved day filters:")
			for name, q := range cfg.SavedDayFilters {
				fmt.Printf("  %-20s  %s\n", name, q)
			}
			return
		}

		if filterDaysDelete != "" {
			if _, ok := cfg.SavedDayFilters[filterDaysDelete]; !ok {
				log.Fatalf("Filter '%s' not found.", filterDaysDelete)
			}
			delete(cfg.SavedDayFilters, filterDaysDelete)
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatalf("Failed to save config: %v", err)
			}
			fmt.Printf("Filter '%s' deleted.\n", filterDaysDelete)
			return
		}

		if saveDaysFilter == "" {
			fmt.Println("No action taken. Use --save <n> [query], --list, or --delete <n>.")
			return
		}

		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		if cfg.SavedDayFilters == nil {
			cfg.SavedDayFilters = make(map[string]string)
		}
		cfg.SavedDayFilters[saveDaysFilter] = query
		if err := config.SaveConfig(cfg); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Filter '%s' saved: %q\n", saveDaysFilter, query)
	},
}
