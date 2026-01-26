package cmd

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

type CalConfig struct {
	ShowDay    int
	Year       int
	Month      int
	Force      bool
	FilterName string

	Padding  int
	Monday   bool
	NoColor  bool
	Vertical bool
	YearView bool
}

type YearMonth struct {
	Year  int
	Month time.Month
}

type DayInfo struct {
	Date         time.Time
	Hours        float64
	IsWeekend    bool
	IsHoliday    bool
	IsToday      bool
	Appointments []mantis.TimesheetsResponse
}

var calCfg CalConfig

var mesesLong = [...]string{
	"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
	"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
}

const (
	INVERT = "\033[7m"
	BOLD   = "\033[1m"
	GREEN  = "\033[32m"
	RED    = "\033[31m"
	CYAN   = "\033[36m"
	BLUE   = "\033[34m"
	RESET  = "\033[0m"
)

type RGB struct{ R, G, B uint8 }

func hex2rgb(hex string) RGB {
	if len(hex) == 7 && hex[0] == '#' {
		hex = hex[1:]
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return RGB{0, 0, 0}
	}
	return RGB{r, g, b}
}

func lerp(a, b RGB, t float64) RGB {
	return RGB{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

var (
	Red    = hex2rgb("#ea6962")
	Yellow = hex2rgb("#e78a4e")
	Green  = hex2rgb("#a9b665")
)

type Renderer struct {
	Padding  int
	Monday   bool
	NoColor  bool
	Vertical bool
	Now      time.Time
}

func NewRenderer(cfg CalConfig) Renderer {
	return Renderer{
		Padding:  cfg.Padding,
		Monday:   cfg.Monday,
		NoColor:  cfg.NoColor,
		Vertical: cfg.Vertical,
		Now:      time.Now(),
	}
}

func (r Renderer) colorForHours(journeyHours, dayHours float64) string {
	if r.NoColor {
		return ""
	}

	if journeyHours <= 0 {
		journeyHours = 8
	}

	h := math.Max(0, math.Min(dayHours, journeyHours))

	t := h / journeyHours

	var c RGB
	if t <= 0.5 {
		local := t / 0.5
		c = lerp(Red, Yellow, local)
	} else {
		local := (t - 0.5) / 0.5
		c = lerp(Yellow, Green, local)
	}

	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

func (r Renderer) RenderMonth(
	year int,
	month time.Month,
	days []DayInfo,
	journeyHours float64,
) {
	r.printHeader(year, month)

	if r.Vertical {
		r.renderVertical(days, journeyHours)
	} else {
		r.printWeekdays()
		r.renderHorizontal(days, journeyHours)
	}
}

func (r Renderer) printHeader(year int, month time.Month) {
	title := fmt.Sprintf("%s %d", mesesLong[int(month)-1], year)

	cellWidth := 2 + r.Padding
	totalWidth := 7 * cellWidth

	if len(title) < totalWidth {
		leftPad := (totalWidth - len(title)) / 2
		fmt.Printf("%s%s\n", strings.Repeat(" ", leftPad), title)
	} else {
		fmt.Println(title)
	}
}

func (r Renderer) printWeekdays() {
	labels := []string{"do", "se", "te", "qu", "qu", "se", "sá"}
	if r.Monday {
		labels = []string{"se", "te", "qu", "qu", "se", "sá", "do"}
	}

	for _, l := range labels {
		fmt.Printf("%-*s", 2+r.Padding, l)
	}
	fmt.Println()
}

func (r Renderer) renderHorizontal(days []DayInfo, journeyHours float64) {
	if len(days) == 0 {
		return
	}

	first := days[0].Date
	offset := int(first.Weekday())
	if r.Monday {
		offset = (offset + 6) % 7
	}

	cellWidth := 2 + r.Padding
	fmt.Print(strings.Repeat(" ", offset*cellWidth))

	weekStart := time.Sunday
	if r.Monday {
		weekStart = time.Monday
	}

	for i, d := range days {
		if i > 0 && d.Date.Weekday() == weekStart {
			fmt.Println()
		}
		fmt.Print(r.formatDay(d, journeyHours))
	}

	fmt.Println()
}

func (r Renderer) renderVertical(days []DayInfo, journeyHours float64) {
	if len(days) == 0 {
		return
	}

	cellWidth := 2 + r.Padding

	labels := []string{"do", "se", "te", "qu", "qu", "se", "sá"}
	if r.Monday {
		labels = []string{"se", "te", "qu", "qu", "se", "sá", "do"}
	}

	rows := make([][]DayInfo, 7)

	first := days[0].Date
	offset := int(first.Weekday())
	if r.Monday {
		offset = (offset + 6) % 7
	}

	for i := 0; i < offset; i++ {
		rows[i] = append(rows[i], DayInfo{})
	}

	for _, d := range days {
		wd := int(d.Date.Weekday())
		if r.Monday {
			wd = (wd + 6) % 7
		}
		rows[wd] = append(rows[wd], d)
	}

	for i := range 7 {
		fmt.Printf("%-3s", labels[i])

		for _, d := range rows[i] {
			if d.Date.IsZero() {
				fmt.Print(strings.Repeat(" ", cellWidth))
				continue
			}
			fmt.Print(r.formatDay(d, journeyHours))
		}
		fmt.Println()
	}
}

func (r Renderer) formatDay(d DayInfo, journeyHours float64) string {
	reset := RESET
	bold := BOLD
	invert := INVERT
	if r.NoColor {
		bold = ""
	}

	color := ""
	if !r.NoColor {
		switch {
		case d.Hours > 0:
			color = r.colorForHours(journeyHours, d.Hours)
		case d.IsHoliday:
			color = CYAN
		case d.IsToday:
			color = RED
		case !d.IsWeekend && d.Date.Before(r.Now):
			color = RED
		case d.IsWeekend:
			color = BLUE
		}
	}

	cell := fmt.Sprintf("%02d", d.Date.Day())

	if d.IsToday {
		return fmt.Sprintf("%s%s%s%s%s%s",
			bold,
			invert,
			color,
			cell,
			reset,
			strings.Repeat(" ", r.Padding),
		)
	}

	return fmt.Sprintf("%s%s%s%s%s",
		bold,
		color,
		cell,
		reset,
		strings.Repeat(" ", r.Padding),
	)
}

func (r Renderer) RenderDay(
	year int,
	month time.Month,
	day int,
	info DayInfo,
	nonBusiness map[int]mantis.NonBusinessDay,
) {
	fmt.Printf(
		"\n%s--- Apontamentos para %d de %s %d ---%s\n",
		BOLD, day, mesesLong[int(month)-1], year, RESET,
	)

	if len(info.Appointments) == 0 {
		if nbd, ok := nonBusiness[day]; ok {
			fmt.Printf("%sDia não útil: %s%s\n", CYAN, nbd.Name, RESET)
		} else {
			fmt.Printf("%sApontamentos não encontrados.%s\n", CYAN, RESET)
		}
		return
	}

	fmt.Printf("%s%-8s %-40s %-10s%s\n", BOLD, "Hours", "Description", "Ticket", RESET)
	fmt.Println(strings.Repeat("-", 70))

	for _, ts := range info.Appointments {
		fmt.Printf("%-8.2f %-40.40s %-10s\n", ts.Quantity, ts.Description, ts.TicketNo)
	}

	fmt.Println(strings.Repeat("-", 70))
}

func isWeekendDay(day time.Weekday) bool {
	return day == time.Saturday || day == time.Sunday
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() &&
		a.Month() == b.Month() &&
		a.Day() == b.Day()
}

func buildDays(
	year int,
	month time.Month,
	hoursByDate map[string]float64,
	dateAppointments map[string][]mantis.TimesheetsResponse,
	nonBusiness map[int]mantis.NonBusinessDay,
	now time.Time,
) []DayInfo {

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)

	var days []DayInfo
	for d := 1; d <= last.Day(); d++ {
		date := time.Date(year, month, d, 0, 0, 0, 0, time.Local)
		key := date.Format("2006-01-02")

		days = append(days, DayInfo{
			Date:         date,
			Hours:        hoursByDate[key],
			IsWeekend:    isWeekendDay(date.Weekday()),
			IsHoliday:    nonBusiness[d].Name != "",
			IsToday:      sameDay(date, now),
			Appointments: dateAppointments[key],
		})
	}

	return days
}

func init() {
	now := time.Now()

	calCmd.Flags().IntVarP(&calCfg.ShowDay, "day", "d", 0, "Show a specific day")
	calCmd.Flags().IntVar(&calCfg.Year, "year", now.Year(), "Year")
	calCmd.Flags().IntVar(&calCfg.Month, "month", int(now.Month()), "Month (1-12)")
	calCmd.Flags().BoolVarP(&calCfg.Force, "force", "f", false, "Force refresh")
	calCmd.Flags().StringVarP(&calCfg.FilterName, "filter", "F", "", "Saved filter")
	calCmd.Flags().BoolVarP(&calCfg.YearView, "year-view", "y", false, "Show whole year")

	calCmd.Flags().IntVarP(&calCfg.Padding, "padding", "j", 1, "Horizontal padding")
	calCmd.Flags().BoolVar(&calCfg.Monday, "monday", false, "Week starts on Monday")
	calCmd.Flags().BoolVar(&calCfg.NoColor, "no-color", false, "Disable colors")
	calCmd.Flags().BoolVarP(&calCfg.Vertical, "vertical", "M", false, "Vertical (ncal-style)")

	calCmd.RegisterFlagCompletionFunc("filter", filterNameCompletionFunc)
	rootCmd.AddCommand(calCmd)
}

var calCmd = &cobra.Command{
	Use:   "cal",
	Short: "Show a calendar with your appointments highlighted",
	Run: func(cmd *cobra.Command, args []string) {
		currentProfileName := appConfig.DefaultProfile
		if profileName != "" {
			currentProfileName = profileName
		}
		profile := appConfig.Profiles[currentProfileName]
		r := NewRenderer(calCfg)

		if calCfg.YearView {
			timesheetsByMonth := map[YearMonth][]mantis.TimesheetsResponse{}
			nonBusinessByMonth := map[YearMonth]map[int]mantis.NonBusinessDay{}

			for m := time.January; m <= time.December+1; m++ {
				realMonth := m
				realYear := calCfg.Year
				if m > time.December {
					realMonth = time.January
					realYear = calCfg.Year + 1
				}

				tsFilename := timesheetCacheKey(currentUserID, realYear, realMonth)
				var ts []mantis.TimesheetsResponse
				var err error

				if !calCfg.Force {
					ts, err = cache.ReadFromCache[mantis.TimesheetsResponse](tsFilename)
					if err != nil {
						log.Printf("Warning: failed to read timesheet cache (%s): %v", tsFilename, err)
					}
				}

				if calCfg.Force || len(ts) == 0 {
					ts, err = mantisClient.Timesheet.GetTimesheets(
						mantisCtx,
						currentUserID,
						realYear,
						realMonth,
					)
					if err != nil {
						log.Fatalf("Error getting timesheets: %v", err)
					}
					if err := cache.WriteToCache(tsFilename, ts); err != nil {
						log.Printf("Warning: failed to write timesheet cache (%s): %v", tsFilename, err)
					}
				}

				if calCfg.FilterName != "" {
					f := appConfig.SavedFilters[calCfg.FilterName]
					ts = utils.ApplyFilter(ts, f, profile)
				}

				ym := YearMonth{Year: realYear, Month: realMonth}

				timesheetsByMonth[ym] = ts

				nbFilename := nonBusinessCacheKey(realYear, realMonth)
				var nbResp []mantis.NonBusinessDay
				if !calCfg.Force {
					nbResp, err = cache.ReadFromCache[mantis.NonBusinessDay](nbFilename)
					if err != nil {
						log.Printf("Warning: failed to read non-business cache (%s): %v", nbFilename, err)
						nbResp = nil
					}
				}

				if calCfg.Force || len(nbResp) == 0 {
					fetched, err := mantisClient.Calendar.GetNonBusinessDays(mantisCtx, realYear, realMonth)
					if err != nil {
						log.Printf("Warning: failed to get non-business days for %04d-%02d: %v", realYear, realMonth, err)
						nbResp = nil
					} else {
						nbResp = fetched
						if err := cache.WriteToCache(nbFilename, nbResp); err != nil {
							log.Printf("Warning: failed to write non-business cache (%s): %v", nbFilename, err)
						}
					}
				}

				nbMap := map[int]mantis.NonBusinessDay{}
				for _, d := range nbResp {
					nbMap[d.Date.Day()] = d
				}
				nonBusinessByMonth[ym] = nbMap
			}

			allDays := map[time.Month][]DayInfo{}
			hoursByDateByMonth := map[YearMonth]map[string]float64{}
			dateAppointmentsByMonth := map[YearMonth]map[string][]mantis.TimesheetsResponse{}

			for m := time.January; m <= time.December+1; m++ {
				realMonth := m
				realYear := calCfg.Year
				if m > time.December {
					realMonth = time.January
					realYear = calCfg.Year + 1
				}

				ym := YearMonth{Year: realYear, Month: realMonth}

				hoursByDateByMonth[ym] = map[string]float64{}
				dateAppointmentsByMonth[ym] = map[string][]mantis.TimesheetsResponse{}
			}

			for m := time.January; m <= time.December+1; m++ {
				realMonth := m
				realYear := calCfg.Year

				if m > time.December {
					realMonth = time.January
					realYear = calCfg.Year + 1
				}

				srcYM := YearMonth{Year: realYear, Month: realMonth}

				for _, ts := range timesheetsByMonth[srcYM] {
					parsedDate, err := time.Parse(time.RFC3339, ts.DateDoc)
					if err != nil {
						continue
					}

					key := parsedDate.Format("2006-01-02")
					dstYM := YearMonth{
						Year:  parsedDate.Year(),
						Month: parsedDate.Month(),
					}

					if _, ok := hoursByDateByMonth[dstYM]; ok {
						hoursByDateByMonth[dstYM][key] += ts.Quantity
						dateAppointmentsByMonth[dstYM][key] =
							append(dateAppointmentsByMonth[dstYM][key], ts)
					}
				}
			}

			for m := time.January; m <= time.December; m++ {
				ym := YearMonth{Year: calCfg.Year, Month: m}

				allDays[m] = buildDays(
					calCfg.Year,
					m,
					hoursByDateByMonth[ym],
					dateAppointmentsByMonth[ym],
					nonBusinessByMonth[ym],
					time.Now(),
				)
			}

			r.RenderYear(calCfg.Year, allDays, profile.DailyJourney)
			return
		}

		filename := timesheetCacheKey(
			currentUserID,
			calCfg.Year,
			time.Month(calCfg.Month),
		)

		var err error
		var timesheets []mantis.TimesheetsResponse

		if !calCfg.Force {
			timesheets, err = cache.ReadFromCache[mantis.TimesheetsResponse](filename)
			if err != nil {
				log.Printf("Warning: failed to read timesheet cache (%s): %v", filename, err)
				timesheets = nil
			}
		}

		if calCfg.Force || len(timesheets) == 0 {
			timesheets, err = mantisClient.Timesheet.GetTimesheets(
				mantisCtx,
				currentUserID,
				calCfg.Year,
				time.Month(calCfg.Month),
			)
			if err != nil {
				log.Fatalf("Error getting timesheets: %v", err)
			}
			if err := cache.WriteToCache(filename, timesheets); err != nil {
				log.Printf("Warning: Failed to write to cache: %v", err)
			}
		}

		if calCfg.FilterName != "" {
			f := appConfig.SavedFilters[calCfg.FilterName]
			timesheets = utils.ApplyFilter(timesheets, f, profile)
		}

		nbFilename := nonBusinessCacheKey(calCfg.Year, time.Month(calCfg.Month))
		var nonBusinessResp []mantis.NonBusinessDay
		if !calCfg.Force {
			nonBusinessResp, err = cache.ReadFromCache[mantis.NonBusinessDay](nbFilename)
			if err != nil {
				log.Printf("Warning: failed to read non-business cache (%s): %v", nbFilename, err)
				nonBusinessResp = nil
			}
		}
		if calCfg.Force || len(nonBusinessResp) == 0 {
			nb, err := mantisClient.Calendar.GetNonBusinessDays(
				mantisCtx,
				calCfg.Year,
				time.Month(calCfg.Month),
			)
			if err != nil {
				log.Printf("Warning: failed to get non-business days: %v", err)
				nonBusinessResp = nil
			} else {
				nonBusinessResp = nb
				if err := cache.WriteToCache(nbFilename, nonBusinessResp); err != nil {
					log.Printf("Warning: failed to write non-business cache (%s): %v", nbFilename, err)
				}
			}
		}

		nonBusiness := map[int]mantis.NonBusinessDay{}
		for _, d := range nonBusinessResp {
			nonBusiness[d.Date.Day()] = d
		}

		hoursByDate := map[string]float64{}
		dateAppointments := map[string][]mantis.TimesheetsResponse{}

		for _, ts := range timesheets {
			parsedDate, err := time.Parse(time.RFC3339, ts.DateDoc)
			if err != nil {
				continue
			}

			if inCalendarMonth(parsedDate, calCfg.Year, time.Month(calCfg.Month)) {
				key := parsedDate.Format("2006-01-02")
				hoursByDate[key] += ts.Quantity
				dateAppointments[key] = append(dateAppointments[key], ts)
			}
		}

		days := buildDays(
			calCfg.Year,
			time.Month(calCfg.Month),
			hoursByDate,
			dateAppointments,
			nonBusiness,
			time.Now(),
		)

		if calCfg.ShowDay != 0 {
			info := days[calCfg.ShowDay-1]
			r.RenderDay(
				calCfg.Year,
				time.Month(calCfg.Month),
				calCfg.ShowDay,
				info,
				nonBusiness,
			)
			return
		}

		r.RenderMonth(
			calCfg.Year,
			time.Month(calCfg.Month),
			days,
			profile.DailyJourney,
		)
	},
}

func (r Renderer) RenderYear(
	year int,
	allDays map[time.Month][]DayInfo,
	journeyHours float64,
) {
	monthsPerRow := 3
	cellWidth := 2 + r.Padding
	monthWidth := cellWidth * 7

	fmt.Printf("%*d\n\n", monthWidth*monthsPerRow/2, year)

	for row := range 4 {
		start := row*monthsPerRow + 1
		end := start + monthsPerRow - 1

		for m := start; m <= end; m++ {
			title := mesesLong[m-1]
			fmt.Printf("%-*s", monthWidth, center(title, monthWidth))
		}
		fmt.Println()

		for m := start; m <= end; m++ {
			r.printWeekdaysInline(monthWidth)
		}
		fmt.Println()

		monthRows := make([][][]string, monthsPerRow)
		maxRows := 0

		for i, m := 0, start; m <= end; m, i = m+1, i+1 {
			monthRows[i] = r.buildMonthGrid(allDays[time.Month(m)], journeyHours)
			if len(monthRows[i]) > maxRows {
				maxRows = len(monthRows[i])
			}
		}

		for rrow := 0; rrow < maxRows; rrow++ {
			for i := range monthsPerRow {
				if rrow < len(monthRows[i]) {
					fmt.Printf("%-*s", monthWidth, strings.Join(monthRows[i][rrow], ""))
				} else {
					fmt.Printf("%-*s", monthWidth, "")
				}
			}
			fmt.Println()
		}

		fmt.Println()
	}
}

func (r Renderer) printWeekdaysInline(width int) {
	labels := []string{"do", "se", "te", "qu", "qu", "se", "sá"}
	if r.Monday {
		labels = []string{"se", "te", "qu", "qu", "se", "sá", "do"}
	}

	cellWidth := 2 + r.Padding

	var b strings.Builder
	for _, l := range labels {
		fmt.Fprintf(&b, "%-*s", cellWidth, l)
	}

	fmt.Printf("%-*s", width, b.String())
}

func (r Renderer) buildMonthGrid(days []DayInfo, journeyHours float64) [][]string {
	if len(days) == 0 {
		return nil
	}

	cellWidth := 2 + r.Padding
	first := days[0].Date

	offset := int(first.Weekday())
	if r.Monday {
		offset = (offset + 6) % 7
	}

	var rows [][]string
	row := make([]string, 0, 7)

	for i := 0; i < offset; i++ {
		row = append(row, strings.Repeat(" ", cellWidth))
	}

	for _, d := range days {
		if len(row) == 7 {
			rows = append(rows, row)
			row = []string{}
		}

		row = append(row, r.formatDay(d, journeyHours))
	}

	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, strings.Repeat(" ", cellWidth))
		}
		rows = append(rows, row)
	}

	return rows
}
func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	return strings.Repeat(" ", left) + s
}

func timesheetCacheKey(userID int, year int, month time.Month) string {
	return fmt.Sprintf(cache.TimesheetsCacheFileName, userID, year, int(month))
}

func nonBusinessCacheKey(year int, month time.Month) string {
	return fmt.Sprintf(cache.NonBusinessCacheFileName, year, int(month))
}

func inCalendarMonth(t time.Time, year int, month time.Month) bool {
	start := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	end := start.AddDate(0, 1, 0)
	return !t.Before(start) && t.Before(end)
}
