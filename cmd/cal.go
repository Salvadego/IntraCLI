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
	RESET  = "\033[0m"
)

var (
	mesesLong = [...]string{
		"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
	}
	showDay int
)

func init() {
	calCmd.Flags().IntVarP(&showDay, "day", "d", 0, "Show appointments for a specific day of the month")
	rootCmd.AddCommand(calCmd)
}

var calCmd = &cobra.Command{
	Use:   "cal",
	Short: "Show a calendar with your appointments highlighted",
	Long: `Retrieves your timesheet entries (appointments) and displays them
on a calendar-like view for the current month.`,
	Run: func(cmd *cobra.Command, args []string) {
		filename := fmt.Sprintf(cache.TimesheetsCacheFileName, currentUserID)
		timesheets, err := mantisClient.Timesheet.GetTimesheets(
			mantisCtx,
			currentUserID,
		)
		if err != nil {
			log.Fatalf("Error getting timesheets for calendar: %v", err)
		}
		err = cache.WriteToCache(filename, timesheets)
		if err != nil {
			log.Printf("Warning: Failed to write timesheets to cache: %v", err)
		}

		hoursByDay := make(map[int]float64)
		dailyAppointments := make(map[int][]mantis.TimesheetsResponse)

		now := time.Now()
		currentYear := now.Year()
		currentMonth := now.Month()

		for _, ts := range timesheets {
			parsedDate, err := time.Parse("2006-01-02T15:04:05Z", ts.DateDoc)
			if err != nil {
				log.Printf("Warning: Could not parse date '%s': %v", ts.DateDoc, err)
				continue
			}

			if parsedDate.Year() == currentYear && parsedDate.Month() == currentMonth {
				day := parsedDate.Day()
				hoursByDay[day] += ts.Quantity
				dailyAppointments[day] = append(dailyAppointments[day], ts)
			}
		}

		if showDay != 0 {
			printDailyAppointments(
				currentYear,
				currentMonth,
				showDay,
				dailyAppointments[showDay],
			)
			return
		}

		printCalendar(currentYear, currentMonth, hoursByDay)
	},
}

func printCalendar(year int, month time.Month, hoursByDay map[int]float64) {
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

		totalHours := hoursByDay[day]

		if totalHours >= 8.0 {
			fmt.Printf("%s%s%2d%s ", BOLD, GREEN, day, RESET)
			continue
		}

		if totalHours > 0 {
			fmt.Printf("%s%s%d%s ", BOLD, YELLOW, day, RESET)
			continue
		}

		if isCurrentMonth && day == today.Day() || (!isWeekend && day <= today.Day()) {
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
		fmt.Printf(
			"%sApontamentos não encontrados para: %d de %s.%s\n",
			CYAN,
			day,
			mesesLong[int(month)-1],
			RESET,
		)
		return
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
