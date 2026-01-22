package cmd

import (
	"context"
	"os"
	"time"

	"github.com/Salvadego/mantis/mantis"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var (
	contractID string
	fromStr    string
	toStr      string
)

func init() {
	ticketsCmd.Flags().StringVar(&filterType, "type", "", "Filter by ticket type")
	ticketsCmd.Flags().StringVar(&contractID, "contract", "", "Filter by contract ID")
	ticketsCmd.Flags().StringVar(&fromStr, "from", "", "Filter change date from (RFC3339)")
	ticketsCmd.Flags().StringVar(&toStr, "to", "", "Filter change date to (RFC3339)")

	ticketsCmd.RegisterFlagCompletionFunc(
		"contract",
		contracsCompletion,
	)
	rootCmd.AddCommand(ticketsCmd)
}

var ticketsCmd = &cobra.Command{
	Use:   "tickets",
	Short: "List tickets from dashboard report",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := &mantis.GetReportOptions{}

		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}
		profile, _ := appConfig.Profiles[currentProfileName]
		if filterType == "" {
			filterType = profile.LType
		}

		opts.FilterType = filterType
		opts.FilterContractID = contractID
		opts.FilterUserID = profile.SUserID

		if fromStr != "" {
			t, err := time.Parse(time.RFC3339, fromStr)
			if err != nil {
				return err
			}
			opts.ChangeAtFrom = &t
		}

		if toStr != "" {
			t, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				return err
			}
			opts.ChangeAtTo = &t
		}

		ctx := context.Background()
		tickets, err := mantisClient.Dashboard.GetReport(ctx, opts)
		if err != nil {
			return err
		}

		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting: tw.CellFormatting{AutoWrap: tw.WrapNormal},
					Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
					ColMaxWidths: tw.CellWidth{
						Global: 50,
					},
				},
			}),
		)

		table.Header("Ticket Number", "Status", "Priority", "Description", "SLA")
		for _, t := range tickets {
			table.Append(
				t.TicketNumber,
				t.Status,
				t.Priority,
				t.Description,
				t.PercSLA,
			)
		}

		return table.Render()
	},
}
