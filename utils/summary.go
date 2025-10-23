package utils

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/mantis/mantis"
)

type DailySummary struct {
	Date        string
	Hours       float64
	Status      string
	Project     string
	User        string
	Week        int
	Month       string
	WorkloadBar string
}

func GenerateSummary(timesheets []mantis.TimesheetsResponse, filter types.DailyFilter) (
	[]DailySummary, map[string]float64, map[string]float64,
) {
	filtered := ApplyDailyFilters(timesheets, filter)
	if len(filtered) == 0 {
		return nil, nil, nil
	}

	hoursByDate := groupByDate(filtered)
	summaries := buildSummaries(hoursByDate, filter.MinDailyHours)
	weekly, monthly := aggregateTotals(summaries)

	return summaries, weekly, monthly
}

// -------------------------
//  STAGES
// -------------------------

func groupByDate(timesheets []mantis.TimesheetsResponse) map[string][]mantis.TimesheetsResponse {
	result := make(map[string][]mantis.TimesheetsResponse)
	for _, ts := range timesheets {
		t, err := time.Parse(time.RFC3339, ts.DateDoc)
		if err != nil {
			continue
		}
		date := t.Format("2006-01-02")
		result[date] = append(result[date], ts)
	}
	return result
}

func buildSummaries(grouped map[string][]mantis.TimesheetsResponse, minHours float64) []DailySummary {
	var dates []string
	for d := range grouped {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	var summaries []DailySummary
	for _, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		day := grouped[d]
		hours, project, user := summarizeDay(day)
		status := classify(hours, minHours)
		summaries = append(summaries, DailySummary{
			Date:        d,
			Hours:       hours,
			Status:      status,
			Project:     project,
			User:        user,
			Week:        weekNumber(t),
			Month:       t.Month().String(),
			WorkloadBar: workloadBar(hours, 8),
		})
	}
	return summaries
}

func aggregateTotals(summaries []DailySummary) (map[string]float64, map[string]float64) {
	weekly := make(map[string]float64)
	monthly := make(map[string]float64)
	for _, s := range summaries {
		weekKey := fmt.Sprintf("%s-W%02d", s.Date[:4], s.Week)
		monthKey := fmt.Sprintf("%s-%s", s.Date[:4], s.Month)
		weekly[weekKey] += s.Hours
		monthly[monthKey] += s.Hours
	}
	return weekly, monthly
}

func summarizeDay(timesheets []mantis.TimesheetsResponse) (float64, string, string) {
	var total float64
	var project, user string
	for _, ts := range timesheets {
		total += ts.Quantity
		project = ts.ProjectName
		user = ts.ProjectManager
	}
	return total, project, user
}

func classify(hours, minHours float64) string {
	switch {
	case hours < minHours:
		return "MISSING"
	case hours > 8:
		return "OVERTIME"
	default:
		return "OK"
	}
}

func workloadBar(hours float64, target float64) string {
	units := clamp(int((hours/target)*8), 0, 8)
	return strings.Repeat("█", units) + strings.Repeat("░", 8-units)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func weekNumber(t time.Time) int {
	_, w := t.ISOWeek()
	return w
}
