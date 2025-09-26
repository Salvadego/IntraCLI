package utils

import (
	"regexp"
	"strconv"
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

		if filter.Ticket != "" && !strings.Contains(ts.TicketNo, filter.Ticket) {
			match = false
		}

		if filter.Project != "" {
			aliasInfo, ok := profile.ProjectAliases[filter.Project]
			if !ok {

				match = false
			} else {
				if int(ts.SalesOrder) != aliasInfo.SalesOrder ||
					int(ts.SalesOrderLine) != aliasInfo.SalesOrderLine {
					match = false
				}
			}
		}

		if filter.HasTicketOnly && ts.TicketNo == "" {
			match = false
		}

		if filter.Description != "" {
			re, err := regexp.Compile(filter.Description)
			if err != nil {
				continue
			}
			if !re.MatchString(ts.Description) {
				match = false
			}
		}

		if filter.Quantity != "" {

			re := regexp.MustCompile(`^(>=|<=|>|<|=)(\d+(\.\d+)?)$`)
			matches := re.FindStringSubmatch(filter.Quantity)
			if len(matches) == 0 {
				continue
			}
			operator := matches[1]
			quantityStr := matches[2]

			quantity, err := strconv.ParseFloat(quantityStr, 64)
			if err != nil {
				continue
			}

			switch operator {
			case ">":
				if ts.Quantity <= quantity {
					match = false
				}
			case "<":
				if ts.Quantity >= quantity {
					match = false
				}
			case ">=":
				if ts.Quantity < quantity {
					match = false
				}
			case "<=":
				if ts.Quantity > quantity {
					match = false
				}
			case "=":
				if ts.Quantity != quantity {
					match = false
				}
			default:
				continue
			}
		}

		if filter.Type != "" {
			typeValue, exists := types.TimesheetTypeLookup[filter.Type]
			if !exists || typeValue != ts.TimesheetType {
				match = false
			}
		}

		if match {
			filtered = append(filtered, ts)
		}
	}

	return filtered
}
