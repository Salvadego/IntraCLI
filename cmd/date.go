package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/IntraCLI/utils"
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
	dateSummaryCmd.Flags().StringVar(&dateSummaryFilterName, "filter", "", "Apply a saved day filter")
	dateSummaryCmd.RegisterFlagCompletionFunc("filter", filterDaysNameCompletionFunc)
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
		profile, ok := cfg.Profiles[cfg.DefaultProfile]
		if !ok {
			fmt.Println("Default profile not found in config")
			return
		}

		now := time.Now()
		if calYear == 0 {
			calYear = now.Year()
		}

		timesheets, err := mantisClient.Timesheet.GetTimesheets(
			mantisCtx,
			profile.UserID,
			calYear,
			time.Month(calMonth),
		)
		if err != nil {
			fmt.Printf("Error fetching timesheets: %v\n", err)
			return
		}

		var filter types.DailyFilter
		if dateSummaryFilterName != "" {
			f, ok := cfg.SavedDayFilters[dateSummaryFilterName]
			if !ok {
				fmt.Printf("Daily filter '%s' not found\n", dateSummaryFilterName)
				return
			}
			filter = f
		}

		summaries, weeklyTotals, monthlyTotals := utils.GenerateSummary(timesheets, filter)

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
		for w, h := range weeklyTotals {
			fmt.Printf("  %s: %.2f hours\n", w, h)
		}

		fmt.Println("\nMONTHLY TOTALS:")
		for m, h := range monthlyTotals {
			fmt.Printf("  %s: %.2f hours\n", m, h)
		}

		var total, days float64
		for _, s := range summaries {
			total += s.Hours
			days++
		}
		if days > 0 {
			avg := total / days
			fmt.Printf("\nAVERAGE DAILY HOURS: %.2f (target %.2f)\n", avg, filter.MinDailyHours)
		}
	},
}
