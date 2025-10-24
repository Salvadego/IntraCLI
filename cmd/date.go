package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var (
	dateSummaryFilterName string
)

func init() {
	dateSummaryCmd.Flags().StringVar(&dateSummaryFilterName, "filter-day", "", "Apply a saved day filter")
	dateSummaryCmd.Flags().StringVar(&filterName, "filter-timesheet", "", "Apply a saved timesheet filter")

	dateSummaryCmd.RegisterFlagCompletionFunc("filter-day", filterDaysNameCompletionFunc)
	dateSummaryCmd.RegisterFlagCompletionFunc("filter-timesheet", filterNameCompletionFunc)

	dateSummaryCmd.Flags().IntVarP(&calYear, "year", "y", 0, "Year (default current)")
	dateSummaryCmd.Flags().IntVarP(&calMonth, "month", "m", 0, "Month (1-12, optional)")
	rootCmd.AddCommand(dateSummaryCmd)
}

var dateSummaryCmd = &cobra.Command{
	Use:   "date-summary",
	Short: "Show sorted, colored, and aggregated daily summary",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.InitializeConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}
		profile, profileExists := appConfig.Profiles[currentProfileName]
		if !profileExists {
			log.Fatalf("Profile '%s' not found", currentProfileName)
		}

		now := time.Now()
		if calYear == 0 {
			calYear = now.Year()
		}

		var timesheets []mantis.TimesheetsResponse
		if calMonth == 0 {
			var all []mantis.TimesheetsResponse
			for m := time.January; m <= time.December; m++ {
				tss, err := mantisClient.Timesheet.GetTimesheets(
					mantisCtx,
					currentUserID,
					calYear,
					m)
				if err != nil {
					fmt.Printf("Error fetching month %d: %v\n", m, err)
					continue
				}
				all = append(all, tss...)
			}
			timesheets = all
		} else {
			timesheets, err = mantisClient.Timesheet.GetTimesheets(mantisCtx,
				currentUserID,
				calYear,
				time.Month(calMonth))
			if err != nil {
				fmt.Printf("Error fetching timesheets: %v\n", err)
				return
			}
		}

		if err != nil {
			fmt.Printf("Error fetching timesheets: %v\n", err)
			return
		}

		var timesheetFilter types.TimesheetFilter
		if filterName != "" {
			f, ok := cfg.SavedFilters[filterName]
			if !ok {
				fmt.Printf("Timesheet filter '%s' not found\n", filterName)
				return
			}
			timesheetFilter = f
		}

		timesheets = utils.ApplyFilter(timesheets, timesheetFilter, profile)

		var dayFilter types.DailyFilter
		if dateSummaryFilterName != "" {
			f, ok := cfg.SavedDayFilters[dateSummaryFilterName]
			if !ok {
				fmt.Printf("Daily filter '%s' not found\n", dateSummaryFilterName)
				return
			}
			dayFilter = f
		}

		summaries, weeklyTotals, monthlyTotals := utils.GenerateSummary(timesheets, profile, dayFilter)

		colorCfg := renderer.ColorizedConfig{
			Header: renderer.Tint{FG: renderer.Colors{color.FgHiWhite, color.Bold}},
			Column: renderer.Tint{
				Columns: []renderer.Tint{
					{}, {}, {}, {}, {}, {}, {}, {},
				},
			},
		}

		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithRenderer(renderer.NewColorized(colorCfg)),
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
					Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
				},
			}),
		)

		table.Header("DATE", "HOURS", "STATUS", "PROJECT", "USER", "WEEK", "MONTH", "WORKLOAD")

		for _, s := range summaries {
			var statusColor *color.Color
			switch s.Status {
			case "MISSING":
				statusColor = color.New(color.FgHiRed)
			case "OVERTIME":
				statusColor = color.New(color.FgHiYellow)
			default:
				statusColor = color.New(color.FgHiGreen)
			}

			table.Append([]any{
				s.Date,
				fmt.Sprintf("%.2f", s.Hours),
				statusColor.Sprint(s.Status),
				s.Project,
				s.User,
				fmt.Sprintf("%d", s.Week),
				s.Month,
				s.WorkloadBar,
			})
		}

		table.Render()

		fmt.Println("\nWEEKLY TOTALS:")
		weeks := make([]string, 0, len(weeklyTotals))
		for w := range weeklyTotals {
			weeks = append(weeks, w)
		}
		sort.Strings(weeks)

		for _, w := range weeks {
			fmt.Printf("  %s: %.2f hours\n", w, weeklyTotals[w])
		}

		printMonthlyTotals(monthlyTotals)

		var total, days float64
		for _, s := range summaries {
			total += s.Hours
			days++
		}
		if days > 0 {
			avg := total / days
			fmt.Printf("\nAVERAGE DAILY HOURS: %.2f (target %.2f)\n", avg, dayFilter.MinDailyHours)
		}
	},
}

func printMonthlyTotals(monthlyTotals map[string]float64) {
	type kv struct {
		key string
		t   time.Time
	}

	items := make([]kv, 0, len(monthlyTotals))
	for k := range monthlyTotals {
		t, err := time.Parse("2006-January", k)
		if err != nil {
			continue
		}
		items = append(items, kv{key: k, t: t})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].t.Before(items[j].t)
	})

	fmt.Println("\nMONTHLY TOTALS:")
	for _, it := range items {
		fmt.Printf("  %s: %.2f hours\n", it.key, monthlyTotals[it.key])
	}
}
