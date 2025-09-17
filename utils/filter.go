package utils

import (
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/mantis/mantis"
)

func ApplyFilter(timesheets []mantis.TimesheetsResponse, filter types.TimesheetFilter, profile config.Profile) []mantis.TimesheetsResponse {
	var filtered []mantis.TimesheetsResponse

	for _, ts := range timesheets {
		match := true

		parsedDate, err := time.Parse(time.RFC3339, ts.DateDoc)
		if err != nil {
			continue
		}

		// Date range
		if filter.FromDate != "" {
			from, _ := time.Parse("2006-01-02", filter.FromDate)
			if parsedDate.Before(from) {
				match = false
			}
		}
		if filter.ToDate != "" {
			to, _ := time.Parse("2006-01-02", filter.ToDate)
			if parsedDate.After(to) {
				match = false
			}
		}

		// Ticket
		if filter.Ticket != "" && !strings.Contains(ts.TicketNo, filter.Ticket) {
			match = false
		}

		// Project
		if filter.Project != "" {
			aliasInfo, ok := profile.ProjectAliases[filter.Project]
			if !ok {
				// alias inexistente -> não bate nada
				match = false
			} else {
				if int(ts.SalesOrder) != aliasInfo.SalesOrder ||
					int(ts.SalesOrderLine) != aliasInfo.SalesOrderLine {
					match = false
				}
			}
		}

		// Only with ticket
		if filter.HasTicketOnly && ts.TicketNo == "" {
			match = false
		}

		if match {
			filtered = append(filtered, ts)
		}
	}

	return filtered
}
