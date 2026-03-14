package utils

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/config"
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

func GenerateSummary(
	timesheets []mantis.TimesheetsResponse,
	prof config.Profile,
	dayFilterQuery string,
	minDailyHours float64,
) ([]DailySummary, map[string]float64, map[string]float64) {
	if len(timesheets) == 0 {
		return nil, nil, nil
	}

	hoursByDate := groupByDate(timesheets)
	summaries := buildSummaries(hoursByDate, prof.DailyJourney, minDailyHours)
	summaries = ApplyDailyFilters(summaries, dayFilterQuery)

	weekly, monthly := aggregateTotals(summaries)
	return summaries, weekly, monthly
}

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

func buildSummaries(
	grouped map[string][]mantis.TimesheetsResponse,
	journeyHours, minHours float64,
) []DailySummary {
	dates := make([]string, 0, len(grouped))
	for d := range grouped {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	if journeyHours == 0 {
		journeyHours = 8.0
	}

	var summaries []DailySummary
	for _, d := range dates {
		t, _ := time.Parse("2006-01-02", d)
		hours, project, user := summarizeDay(grouped[d])
		summaries = append(summaries, DailySummary{
			Date:        d,
			Hours:       hours,
			Status:      classify(journeyHours, hours, minHours, 0.001),
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

func summarizeDay(timesheets []mantis.TimesheetsResponse) (hours float64, project, user string) {
	for _, ts := range timesheets {
		hours += ts.Quantity
		project = ts.ProjectName
		user = ts.ProjectManager
	}
	return
}

func classify(journeyHours, hours, minHours, tol float64) string {
	switch {
	case hours < minHours && !ApproxEqual(hours, minHours, tol):
		return "MISSING"
	case hours > journeyHours && !ApproxEqual(hours, journeyHours, tol):
		return "OVERTIME"
	default:
		return "OK"
	}
}

func workloadBar(hours, target float64) string {
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

func ApproxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}
