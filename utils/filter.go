package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/Salvadego/IntraCLI/types"
	"github.com/Salvadego/mantis/mantis"
	"github.com/Salvadego/qlvm"
)

// TimesheetRecord is the qlvm-queryable projection of a mantis timesheet.
// Tags drive the field names available in filter query strings.
//
// Example queries:
//
//	ticket = "INC-123"
//	.ticket = INC          (contains shorthand)
//	#ticket = INC-123      (exact match shorthand)
//	has_ticket?            (exists shorthand)
//	date >= 2025-07-01
//	hours > 4
//	project = myalias
//	type = Normal
type TimesheetRecord struct {
	Ticket      string  `qlvm:"ticket"`
	Hours       float64 `qlvm:"hours"`
	Date        string  `qlvm:"date,date"`
	Description string  `qlvm:"description"`
	Type        string  `qlvm:"type"`
	Project     string  `qlvm:"project"`
	HasTicket   bool    `qlvm:"has_ticket"`

	// idx keeps the original slice position so we can reconstruct the result
	// slice without copying the heavy mantis structs into the record.
	idx int `qlvm:"-"`
}

// DailySummaryRecord is the qlvm-queryable projection of a DailySummary row.
//
// Example queries:
//
//	status = MISSING
//	hours < 4
//	week = 27
//	month = July
//	project = myalias
type DailySummaryRecord struct {
	Date    string  `qlvm:"date,date"`
	Hours   float64 `qlvm:"hours"`
	Status  string  `qlvm:"status"`
	Project string  `qlvm:"project"`
	User    string  `qlvm:"user"`
	Week    int     `qlvm:"week"`
	Month   string  `qlvm:"month"`

	idx int `qlvm:"-"`
}

// Engine is the shared qlvm engine for timesheet queries.
// Initialized at package load — never nil.
var Engine = qlvm.New(
	qlvm.SchemaFromStruct[TimesheetRecord]().
		Prefix('.', "ticket", qlvm.Contains).
		Prefix('#', "description", qlvm.Contains).
		Suffix('?', qlvm.Exists()),
)

// DayEngine is the shared qlvm engine for daily-summary queries.
// Initialized at package load — never nil.
var DayEngine = qlvm.New(
	qlvm.SchemaFromStruct[DailySummaryRecord]().
		Suffix('?', qlvm.Exists()),
)

// newTimesheetRecord converts a mantis response into a queryable record,
// resolving the project alias from the profile when possible.
func newTimesheetRecord(ts mantis.TimesheetsResponse, profile config.Profile, idx int) TimesheetRecord {
	typeName := ts.TimesheetType
	if name, ok := types.TimesheetTypeInverseLookup[ts.TimesheetType]; ok {
		typeName = name
	}

	date := ts.DateDoc
	if len(date) >= 10 {
		date = date[:10]
	}

	return TimesheetRecord{
		Ticket:      ts.TicketNo,
		Hours:       ts.Quantity,
		Date:        date,
		Description: ts.Description,
		Type:        typeName,
		Project:     resolveProject(ts, profile),
		HasTicket:   ts.TicketNo != "",
		idx:         idx,
	}
}

func resolveProject(ts mantis.TimesheetsResponse, profile config.Profile) string {
	for alias, info := range profile.ProjectAliases {
		if int(ts.SalesOrder) == info.SalesOrder &&
			int(ts.SalesOrderLine) == info.SalesOrderLine {
			return alias
		}
	}
	return ts.ProjectName
}

// Apply filters timesheets using a qlvm query string and returns the matching subset.
// An empty query string returns all timesheets unchanged.
func Apply(
	query string,
	timesheets []mantis.TimesheetsResponse,
	profile config.Profile,
) ([]mantis.TimesheetsResponse, error) {
	if query == "" {
		return timesheets, nil
	}

	records := make([]TimesheetRecord, len(timesheets))
	for i, ts := range timesheets {
		records[i] = newTimesheetRecord(ts, profile, i)
	}

	matched, err := qlvm.Filter(Engine, query, records,
		func(r TimesheetRecord) qlvm.Resolver {
			return qlvm.ResolverOf(r)
		},
	)
	if err != nil {
		return nil, err
	}

	result := make([]mantis.TimesheetsResponse, len(matched))
	for i, r := range matched {
		result[i] = timesheets[r.idx]
	}
	return result, nil
}

// ApplyFilter is a convenience wrapper around Apply that swallows the error and
// logs a warning, returning the original slice on failure.  Use Apply when you
// need proper error handling.
func ApplyFilter(
	timesheets []mantis.TimesheetsResponse,
	query string,
	profile config.Profile,
) []mantis.TimesheetsResponse {
	if query == "" {
		return timesheets
	}
	result, err := Apply(query, timesheets, profile)
	if err != nil {
		// Non-fatal: bad query → return unfiltered rather than crashing.
		return timesheets
	}
	return result
}

// ExpandTokens replaces human-friendly time tokens in a raw qlvm query string
// with their concrete values, computed relative to now.
//
// This runs before the query reaches qlvm, so qlvm never sees the tokens.
//
// Supported tokens (case-insensitive):
//
//	Date tokens (expand to YYYY-MM-DD strings):
//	  today         current date
//	  yesterday     one day ago
//	  tomorrow      one day ahead
//	  this-week     Monday of the current ISO week
//	  last-week     Monday of the previous ISO week
//	  next-week     Monday of the next ISO week
//	  this-month    first day of the current month
//	  last-month    first day of the previous month
//	  next-month    first day of the next month
//	  this-year     first day of the current year  (YYYY-01-01)
//	  last-year     first day of the previous year
//	  Nd-ago        N days ago          (e.g. 7d-ago)
//	  Nw-ago        N weeks ago         (e.g. 2w-ago)
//	  Nm-ago        N months ago        (e.g. 3m-ago)
//	  Nd-from-now   N days from now
//	  Nw-from-now   N weeks from now
//	  Nm-from-now   N months from now
//
//	Numeric / string tokens (useful for week/month/year fields):
//	  this-week-num   current ISO week number  (e.g. 11)
//	  last-week-num   previous ISO week number
//	  this-month-name current month name       (e.g. March)
//	  last-month-name previous month name
//	  this-year-num   current year             (e.g. 2026)
//	  last-year-num   previous year
//
// Examples:
//
//	"date >= yesterday"                   → "date >= 2026-03-11"
//	"date >= this-month AND date < today" → "date >= 2026-03-01 AND date < 2026-03-12"
//	"date >= 7d-ago"                      → "date >= 2026-03-05"
//	"week = this-week-num"                → "week = 11"
//	"month = this-month-name"             → "month = March"
func ExpandTokens(query string) string {
	now := time.Now()
	date := func(t time.Time) string { return t.Format("2006-01-02") }

	// Monday of the week containing t.
	monday := func(t time.Time) time.Time {
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7 // Sunday → 7
		}
		return t.AddDate(0, 0, -(wd - 1))
	}

	// Fixed tokens: longest first so e.g. "last-week-num" beats "last-week".
	fixed := []struct{ token, value string }{
		// Numeric / name tokens
		{"this-week-num", fmt.Sprintf("%d", isoWeek(now))},
		{"last-week-num", fmt.Sprintf("%d", isoWeek(now.AddDate(0, 0, -7)))},
		{"next-week-num", fmt.Sprintf("%d", isoWeek(now.AddDate(0, 0, 7)))},
		{"this-month-name", now.Month().String()},
		{"last-month-name", now.AddDate(0, -1, 0).Month().String()},
		{"next-month-name", now.AddDate(0, 1, 0).Month().String()},
		{"this-year-num", fmt.Sprintf("%d", now.Year())},
		{"last-year-num", fmt.Sprintf("%d", now.Year()-1)},
		// Date tokens
		{"this-week", date(monday(now))},
		{"last-week", date(monday(now.AddDate(0, 0, -7)))},
		{"next-week", date(monday(now.AddDate(0, 0, 7)))},
		{"this-month", date(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))},
		{"last-month", date(time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()))},
		{"next-month", date(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()))},
		{"this-year", date(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()))},
		{"last-year", date(time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location()))},
		{"next-year", date(time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location()))},
		{"yesterday", date(now.AddDate(0, 0, -1))},
		{"tomorrow", date(now.AddDate(0, 0, 1))},
		{"today", date(now)},
	}

	out := query
	for _, f := range fixed {
		out = replaceTokenCI(out, f.token, f.value)
	}

	// Relative offsets: Nd-ago, Nw-ago, Nm-ago, Nd-from-now, Nw-from-now, Nm-from-now
	out = reRelative.ReplaceAllStringFunc(out, func(m string) string {
		sub := reRelative.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil {
			return m
		}
		unit := strings.ToLower(sub[2])
		direction := strings.ToLower(sub[3]) // "ago" or "from-now"

		if direction == "ago" {
			n = -n
		}

		var t time.Time
		switch unit {
		case "d":
			t = now.AddDate(0, 0, n)
		case "w":
			t = now.AddDate(0, 0, n*7)
		case "m":
			t = now.AddDate(0, n, 0)
		default:
			return m
		}
		return date(t)
	})

	return out
}

// reRelative matches tokens like 7d-ago, 2w-from-now, 3m-ago.
var reRelative = regexp.MustCompile(`(?i)\b(\d+)([dwm])-(ago|from-now)\b`)

// replaceTokenCI replaces all case-insensitive occurrences of token with value,
// but only when the token appears as a whole word (not inside another word).
func replaceTokenCI(s, token, value string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(token) + `\b`)
	return re.ReplaceAllString(s, value)
}

func isoWeek(t time.Time) int {
	_, w := t.ISOWeek()
	return w
}

// ResolveFilter resolves a --filter flag value into a raw qlvm query string,
// then expands any human-friendly time tokens via ExpandTokens.
//
// Two forms are accepted:
//
//	@name          looks up "name" in the saved map; error if not found
//	anything else  returned as-is (inline qlvm query)
//
// An empty string is a no-op and is returned as "".
//
// Example:
//
//	ResolveFilter(`@toyoMeetings`, cfg.SavedFilters)
//	ResolveFilter(`date >= yesterday AND project = toyo`, …)
var reSavedFilter = regexp.MustCompile(`@([a-zA-Z0-9_\-]+)`)

func ResolveFilter(value string, saved map[string]string) (string, error) {
	if value == "" {
		return "", nil
	}

	var lastErr error

	query := reSavedFilter.ReplaceAllStringFunc(value, func(match string) string {
		name := match[1:]
		if q, ok := saved[name]; ok {
			return q
		}
		lastErr = fmt.Errorf("saved filter %q not found", name)
		return match
	})

	if lastErr != nil {
		return "", lastErr
	}

	return ExpandTokens(query), nil
}

// ResolveFilterOrFatal is a convenience wrapper for cmd code that wants to
// log.Fatal on a bad @name rather than handle the error itself.
func ResolveFilterOrFatal(value string, saved map[string]string) string {
	q, err := ResolveFilter(value, saved)
	if err != nil {
		panic(err.Error())
	}
	return q
}

// ApplyDailyFilters filters a slice of DailySummary rows using a qlvm query string.
// An empty query string returns all summaries unchanged.
func ApplyDailyFilters(summaries []DailySummary, query string) []DailySummary {
	if query == "" {
		return summaries
	}

	records := make([]DailySummaryRecord, len(summaries))
	for i, s := range summaries {
		records[i] = DailySummaryRecord{
			Date:    s.Date,
			Hours:   s.Hours,
			Status:  s.Status,
			Project: s.Project,
			User:    s.User,
			Week:    s.Week,
			Month:   s.Month,
			idx:     i,
		}
	}

	matched, err := qlvm.Filter(DayEngine, query, records,
		func(r DailySummaryRecord) qlvm.Resolver {
			return qlvm.ResolverOf(r)
		},
	)
	if err != nil {
		// Non-fatal: bad query → return unfiltered.
		return summaries
	}

	result := make([]DailySummary, len(matched))
	for i, r := range matched {
		result[i] = summaries[r.idx]
	}
	return result
}
