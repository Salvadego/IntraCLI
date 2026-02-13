package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

func contractsToLines(contracts []mantis.LtContract) []string {
	lines := make([]string, 0, len(contracts))

	for _, c := range contracts {
		lines = append(lines,
			fmt.Sprintf("%-12s  %s", c.ContractID, c.Title),
		)
	}

	return lines
}

func refreshContracts(ctx context.Context) []mantis.LtContract {
	contracts, err := mantisClient.Dashboard.GetReportContracts(ctx)
	if err != nil {
		log.Fatalf("Error getting contracts: %v", err)
	}
	err = cache.WriteToCache(cache.ContractsListCacheFileName, contracts)
	if err != nil {
		log.Fatalf("Failed to write to cache: %v", err)
	}
	return contracts
}

var contractsCmd = &cobra.Command{
	Use:   "contracts",
	Short: "List contracts available for dashboard report",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		contracts, err := cache.ReadFromCache[mantis.LtContract](cache.ContractsListCacheFileName)

		if err != nil || forceTickets {
			refreshContracts(ctx)
		}

		for _, l := range contractsToLines(contracts) {
			fmt.Println(l)
		}
		return nil
	},
}

func init() {
	contractsCmd.Flags().BoolVar(&forceTickets, "force", false, "Refresh contracts response")

	rootCmd.AddCommand(contractsCmd)
}
