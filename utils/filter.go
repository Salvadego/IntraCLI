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

// -------------------------
// TIMESHEET FILTERING
// -------------------------

func ApplyFilter(timesheets []mantis.TimesheetsResponse, filter types.TimesheetFilter, profile config.Profile) []mantis.TimesheetsResponse {
	var filtered []mantis.TimesheetsResponse
	for _, ts := range timesheets {
		if matchTimesheet(ts, filter, profile) {
			filtered = append(filtered, ts)
		}
	}
	return filtered
}

func matchTimesheet(ts mantis.TimesheetsResponse, filter types.TimesheetFilter, profile config.Profile) bool {
	parsedDate, err := time.Parse(time.RFC3339, ts.DateDoc)

	if err != nil {
		return false
	}

	if !dateInRange(parsedDate, filter.FromDate, filter.ToDate) {
		return false
	}

	if filter.Ticket != "" && !strings.Contains(ts.TicketNo, filter.Ticket) {
		return false
	}

	if filter.Project != "" && !matchProject(ts, filter.Project, profile) {
		return false
	}

	if filter.HasTicketOnly && ts.TicketNo == "" {
		return false
	}

	if filter.Description != "" && !matchRegex(ts.Description, filter.Description) {
		return false
	}

	if filter.Quantity != "" && !matchQuantity(ts.Quantity, filter.Quantity) {
		return false
	}

	if filter.Type != "" {
		if typeVal, ok := types.TimesheetTypeLookup[filter.Type]; !ok || typeVal != ts.TimesheetType {
			return false
		}
	}

	return true
}

// -------------------------
// HELPER FUNCTIONS
// -------------------------

func dateInRange(date time.Time, fromStr, toStr string) bool {
	if fromStr != "" {
		from, _ := time.Parse("2006-01-02", fromStr)
		if date.Before(from) {
			return false
		}
	}
	if toStr != "" {
		to, _ := time.Parse("2006-01-02", toStr)
		if date.After(to) {
			return false
		}
	}
	return true
}

func matchProject(ts mantis.TimesheetsResponse, project string, profile config.Profile) bool {
	aliasInfo, ok := profile.ProjectAliases[project]
	if !ok {
		return false
	}
	return int(ts.SalesOrder) == aliasInfo.SalesOrder && int(ts.SalesOrderLine) == aliasInfo.SalesOrderLine
}

func matchRegex(value, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func matchQuantity(quantity float64, expr string) bool {
	re := regexp.MustCompile(`^(>=|<=|>|<|=)(\d+(\.\d+)?)$`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) == 0 {
		return false
	}
	num, _ := strconv.ParseFloat(matches[2], 64)
	switch matches[1] {
	case ">":
		return quantity > num
	case "<":
		return quantity < num
	case ">=":
		return quantity >= num
	case "<=":
		return quantity <= num
	case "=":
		return quantity == num
	}
	return false
}

// -------------------------
// DAILY HOURS FILTER
// -------------------------

func ApplyDailyFilters(timesheets []mantis.TimesheetsResponse, filter types.DailyFilter) []mantis.TimesheetsResponse {
	var result []mantis.TimesheetsResponse

	for _, ts := range timesheets {
		t, err := time.Parse(time.RFC3339, ts.DateDoc)
		if err != nil || !dateInRange(t, filter.FromDate, filter.ToDate) {
			continue
		}
		if filter.Project != "" && !strings.Contains(strings.ToLower(ts.ProjectName), strings.ToLower(filter.Project)) {
			continue
		}
		if filter.User != "" && !strings.Contains(strings.ToLower(ts.ProjectManager), strings.ToLower(filter.User)) {
			continue
		}
		if filter.HasTicketOnly && ts.TicketNo == "" {
			continue
		}
		result = append(result, ts)
	}
	return result
}
