package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/mantis/mantis"
	"github.com/spf13/cobra"
)

// ANSI escape codes:
const (
	BOLD   = "\033[1m"
	GREEN  = "\033[32m"
	RED    = "\033[31m"
	YELLOW = "\033[33m"
	CYAN   = "\033[36m"
	BLUE   = "\033[34m"
	RESET  = "\033[0m"
)

var (
	mesesLong = [...]string{
		"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
	}
	showDay  int
	calYear  int
	calMonth int
	force    bool

	nonBusinessDaysMap = make(map[int]mantis.NonBusinessDay)
	now                = time.Now()
)

func init() {
	calCmd.Flags().IntVarP(&showDay, "day", "d", 0,
		"Show appointments for a specific day of the month")
	calCmd.Flags().IntVar(&calYear, "year", now.Year(), "Year to show")
	calCmd.Flags().IntVar(&calMonth, "month", int(now.Month()), "Month to show (1-12)")
	calCmd.Flags().BoolVarP(&force, "force", "f", false,
		"Force requests instead of using cached.")
	rootCmd.AddCommand(calCmd)
}

var calCmd = &cobra.Command{
	Use:   "cal",
	Short: "Show a calendar with your appointments highlighted",
	Long: `Retrieves your timesheet entries (appointments) from cache and displays them
on a calendar-like view for the current month.`,
	Run: func(cmd *cobra.Command, args []string) {
		filename := fmt.Sprintf(cache.TimesheetsCacheFileName, currentUserID)
		var err error

		var timesheets []mantis.TimesheetsResponse

		if !force {
			timesheets, err = cache.ReadFromCache[mantis.TimesheetsResponse](filename)
			if err != nil {
				log.Printf("Warning: failed to read from cache: %v", err)
			}
		}

		if force || len(timesheets) == 0 {
			timesheets, err = mantisClient.Timesheet.GetTimesheets(
				mantisCtx,
				currentUserID,
				calYear,
				time.Month(calMonth),
			)
			if err != nil {
				log.Fatalf("Error getting timesheets: %v", err)
			}

			adjacentMonths := []time.Time{
				time.Date(calYear, time.Month(calMonth)-1, 1, 0, 0, 0, 0, time.Local),
				time.Date(calYear, time.Month(calMonth)+1, 1, 0, 0, 0, 0, time.Local),
			}
			for _, t := range adjacentMonths {
				ts, err := mantisClient.Timesheet.GetTimesheets(mantisCtx, currentUserID, t.Year(), t.Month())
				if err != nil {
					log.Printf("Warning: Failed to get timesheets for %d-%02d: %v", t.Year(), t.Month(), err)
					continue
				}
				timesheets = append(timesheets, ts...)
			}

			err = cache.WriteToCache(filename, timesheets)
			if err != nil {
				log.Printf("Warning: Failed to write to cache: %v", err)
			}
		}
		nonBusinessDays, err := mantisClient.Calendar.GetNonBusinessDays(
			mantisCtx,
			calYear,
			time.Month(calMonth),
		)

		if err != nil {
			log.Printf("Error getting non-business days: %v\n", err)
		} else {
			for _, day := range nonBusinessDays {
				nonBusinessDaysMap[day.Date.Day()] = day
			}
		}

		hoursByDate := make(map[string]float64)
		dateAppointments := make(map[string][]mantis.TimesheetsResponse)

		currentYear := calYear
		currentMonth := time.Month(calMonth)

		for _, ts := range timesheets {
			parsedDate, err := time.Parse("2006-01-02T15:04:05Z", ts.DateDoc)
			if err != nil {
				log.Printf("Warning: Could not parse date '%s': %v", ts.DateDoc, err)
				continue
			}

			if parsedDate.Year() == currentYear && parsedDate.Month() == currentMonth {
				key := parsedDate.Format("2006-01-02")
				hoursByDate[key] += ts.Quantity
				dateAppointments[key] = append(dateAppointments[key], ts)
			}
		}

		if showDay != 0 {
			date := time.Date(calYear, time.Month(calMonth), showDay, 0, 0, 0, 0, time.Local)
			key := date.Format("2006-01-02")
			printDailyAppointments(
				currentYear,
				currentMonth,
				showDay,
				dateAppointments[key],
			)
			return
		}

		printCalendar(currentYear, currentMonth, hoursByDate)
	},
}

func printCalendar(year int, month time.Month, hoursByDate map[string]float64) {
	fmt.Printf("      %s %d\n", mesesLong[int(month)-1], year)
	fmt.Println("do se te qu qu se sá")

	today := time.Now()
	isCurrentMonth := (today.Year() == year && today.Month() == month)

	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	initialSpaces := int(firstOfMonth.Weekday()) * 3
	fmt.Print(strings.Repeat(" ", initialSpaces))

	for day := 1; day <= lastOfMonth.Day(); day++ {
		currentDay := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		dayOfWeek := currentDay.Weekday()
		isWeekend := isWeekendDay(dayOfWeek)

		if dayOfWeek == time.Sunday && day != 1 {
			fmt.Println()
		}

		key := currentDay.Format("2006-01-02")
		totalHours := hoursByDate[key]

		if totalHours >= 8.0 {
			fmt.Printf("%s%s%2d%s ", BOLD, GREEN, day, RESET)
			continue
		}

		if totalHours > 0 {
			fmt.Printf("%s%s%d%s ", BOLD, YELLOW, day, RESET)
			continue
		}

		if isWeekend && (day <= today.Day() || month < today.Month()) {
			fmt.Printf("%s%2d%s ", BLUE, day, RESET)
			continue
		}

		_, ok := nonBusinessDaysMap[day]

		if ok {
			fmt.Printf("%s%s%2d%s ", BOLD, CYAN, day, RESET)
			continue
		}

		if isCurrentMonth && day == today.Day() {
			fmt.Printf("%s%s%2d%s ", BOLD, RED, day, RESET)
			continue
		}

		if !isWeekend && currentDay.Before(today) {
			fmt.Printf("%s%s%2d%s ", BOLD, RED, day, RESET)
			continue
		}

		fmt.Printf("%2d ", day)
	}
	fmt.Println()
}

func isWeekendDay(day time.Weekday) bool {
	switch day {
	case time.Saturday:
		return true
	case time.Sunday:
		return true
	default:
		return false
	}
}

func printDailyAppointments(
	year int,
	month time.Month,
	day int,
	appointments []mantis.TimesheetsResponse,
) {
	fmt.Printf(
		"\n%s--- Apontamentos para %d de %s %d ---%s\n",
		BOLD,
		day,
		mesesLong[int(month)-1],
		year,
		RESET,
	)

	if len(appointments) == 0 {
		if len(appointments) == 0 {
			if nbd, ok := nonBusinessDaysMap[day]; ok {
				fmt.Printf("%sDia não útil: %s%s\n", CYAN, nbd.Name, RESET)
			} else {
				fmt.Printf("%sApontamentos não encontrados para: %d de %s.%s\n", CYAN, day, mesesLong[int(month)-1], RESET)
			}
			return
		}
	}

	fmt.Printf(
		"%s%-8s %-40s %-10s%s\n",
		BOLD,
		"Hours",
		"Description",
		"Ticket",
		RESET,
	)
	fmt.Println(strings.Repeat("-", 70))

	for _, ts := range appointments {
		fmt.Printf(
			"%-8.2f %-40.40s %-10s\n",
			ts.Quantity,
			ts.Description,
			ts.TicketNo,
		)
	}
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
}
