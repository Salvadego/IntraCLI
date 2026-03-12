package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/muesli/reflow/wrap"

	"os"
	"time"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/Salvadego/IntraCLI/utils"
	"github.com/Salvadego/mantis/mantis"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

type SortBy string

const (
	Created  SortBy = "created"
	Sla      SortBy = "sla"
	Priority SortBy = "priority"
	Status   SortBy = "status"
	Number   SortBy = "number"
)

var sortByValues = []string{
	string(Created),
	string(Sla),
	string(Priority),
	string(Status),
	string(Number),
}

var sortOrderValues = []string{"asc", "desc"}

var (
	contractID                string
	fromStr                   string
	toStr                     string
	shouldDownloadAttachments bool
	sortBy                    string
	sortOrder                 string
	humanDates                bool
	hoursOnly                 bool
	forceTickets              bool
	forceColors               bool
	inline                    bool

	// ticketTypeFilter is the --type flag for the tickets command.
	// It defaults to profile.LType when empty (set at run time, not parse time).
	ticketTypeFilter string
)

func init() {
	ticketsCmd.Flags().StringVar(&ticketTypeFilter, "type", "", "Filter by ticket type (default: profile LType)")
	ticketsCmd.Flags().StringVar(&contractID, "contract", "", "Filter by contract ID")
	ticketsCmd.Flags().StringVar(&fromStr, "from", "", "Filter change date from (RFC3339)")
	ticketsCmd.Flags().StringVar(&toStr, "to", "", "Filter change date to (RFC3339)")
	ticketsCmd.Flags().StringVar(&sortBy, "sort-by", "created", "Sort by: created|sla|priority|status|number")
	ticketsCmd.Flags().StringVar(&sortOrder, "sort-order", "desc", "Sort order: asc|desc")
	ticketsCmd.Flags().BoolVar(&humanDates, "human-dates", false, "Show dates as relative time (e.g. 3d ago)")

	ticketsCmd.Flags().StringVarP(&ticket, "ticket", "t", "", "Inspect ticket details")
	ticketsCmd.Flags().BoolVarP(&shouldDownloadAttachments, "attachment", "a", false, "Download ticket attachments")
	ticketsCmd.Flags().BoolVarP(&hoursOnly, "hoursOnly", "H", false, "Show only project hours")
	ticketsCmd.Flags().BoolVarP(&forceTickets, "force-tickets", "f", false, "Refresh tickets response")
	ticketsCmd.Flags().BoolVarP(&inline, "inline", "i", false, "Display tickets inlined")
	ticketsCmd.Flags().BoolVar(&forceColors, "colors", false, "Force non-TTY colors")

	ticketsCmd.RegisterFlagCompletionFunc(
		"sort-by",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return sortByValues, cobra.ShellCompDirectiveNoFileComp
		},
	)
	ticketsCmd.RegisterFlagCompletionFunc(
		"sort-order",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return sortOrderValues, cobra.ShellCompDirectiveNoFileComp
		},
	)
	ticketsCmd.RegisterFlagCompletionFunc("contract", contracsCompletion)
	ticketsCmd.RegisterFlagCompletionFunc("ticket", ticketCompletionFunc)

	rootCmd.AddCommand(ticketsCmd)
}

var ticketsCmd = &cobra.Command{
	Use:   "tickets",
	Short: "List tickets from dashboard report",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ticket == "" {
			return handleReports()
		}

		resp, err := mantisClient.Dashboard.GetSupportInfo(mantisCtx, ticket)
		if err != nil {
			return err
		}

		if shouldDownloadAttachments && len(resp.Attachments) > 0 {
			for _, att := range resp.Attachments {
				fmt.Printf("Downloading attachment: %s\n", att.FileName)
				resp, err := mantisClient.Dashboard.GetSupportFile(mantisCtx, att)
				if err != nil {
					fmt.Printf("Error trying to download attachment [%s]: %v\n", att.FileName, err)
					continue
				}
				if err = saveFile(att.FileName, resp.FileContent); err != nil {
					fmt.Printf("Error saving attachment [%s]: %v\n", att.FileName, err)
				}
			}
			return nil
		}

		if hoursOnly {
			if resp.TotHrAprovadaPC != "" {
				fmt.Printf("Total Approved Project: %s\n", resp.TotHrAprovadaPC)
				fmt.Printf("Total Consumed Project: %s\n", resp.TotHrPC)
				toHrPc, _ := strconv.ParseFloat(resp.TotHrPC, 64)
				totHrAprovadaPC, _ := strconv.ParseFloat(resp.TotHrAprovadaPC, 64)
				fmt.Printf("Total Disponible Project: %.2f\n", totHrAprovadaPC-toHrPc)
			}
			return nil
		}

		renderTicket(&resp)
		return nil
	},
}

func handleReports() error {
	opts := &mantis.GetReportOptions{}

	currentProfileName := appConfig.DefaultProfile
	if profileName != "" {
		currentProfileName = profileName
	}
	profile, _ := appConfig.Profiles[currentProfileName]

	// --type defaults to the profile's LType when not explicitly set.
	resolvedType := ticketTypeFilter
	if resolvedType == "" {
		resolvedType = profile.LType
	}
	opts.FilterType = resolvedType
	opts.FilterContractID = contractID
	opts.FilterUserID = profile.SUserID

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return err
		}
		opts.ChangeAtFrom = &t
	}

	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return err
		}
		opts.ChangeAtTo = &t
	}

	ctx := context.Background()

	sig := opts.Signature()
	cacheFile := fmt.Sprintf(cache.TicketsCacheFileName, sig)

	tickets, err := cache.ReadFromCache[mantis.TicketResponse](cacheFile)
	if err != nil || forceTickets {
		tickets, err = mantisClient.Dashboard.GetReport(ctx, opts)
		if err != nil {
			log.Fatalf("Error getting tickets: %v", err)
		}
		if err := cache.WriteToCache(cacheFile, tickets); err != nil {
			log.Fatalf("Failed to write to cache: %v", err)
		}
	}

	sortTickets(tickets, SortBy(sortBy), sortOrder)

	if inline {
		for _, l := range ticketsToLines(tickets) {
			fmt.Println(l)
		}
		return nil
	}

	return renderByStatus(tickets)
}

func renderByStatus(tickets []mantis.TicketResponse) error {
	groups := make(map[string][]mantis.TicketResponse)
	for _, t := range tickets {
		groups[t.Status] = append(groups[t.Status], t)
	}

	statuses := make([]string, 0, len(groups))
	for s := range groups {
		statuses = append(statuses, s)
	}
	sort.Strings(statuses)

	for _, status := range statuses {
		fmt.Println()
		utils.SectionStyle.Printf("■ %s ", status)
		utils.MutedStyle.Printf("(%d)\n\n", len(groups[status]))

		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal},
					Alignment:    tw.CellAlignment{Global: tw.AlignLeft},
					ColMaxWidths: tw.CellWidth{Global: 50},
				},
			}),
		)

		table.Header("Ticket Number", "Priority", "Description", "SLA", "Created Date")

		for _, t := range groups[status] {
			created := parseTime(t.TicketCreated).String()
			if humanDates {
				created = humanizeTime(parseTime(t.TicketCreated))
			}
			table.Append(
				t.TicketNumber,
				colorPriority(t.Priority),
				t.Description,
				colorSLA(t.PercSLA),
				created,
			)
		}

		if err := table.Render(); err != nil {
			return err
		}
	}

	return nil
}

func renderTicket(t *mantis.SupportInfoResponse) {
	utils.TitleStyle.Printf("Ticket %s\n", t.ObjectID)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("%s: %s\n", utils.MutedStyle.Sprint("Status"), t.UserStatusDescription)
	fmt.Printf("%s: %s\n", utils.MutedStyle.Sprint("Priority"), colorPriority(t.Priority))
	fmt.Printf("Process Type: %s\n", t.ProcessType)
	fmt.Printf("Category: %s\n", t.CategoryID)
	if t.TotHrAprovadaPC != "" {
		fmt.Printf("Total Approved Project: %s\n", t.TotHrAprovadaPC)
		fmt.Printf("Total Consumed Project: %s\n", t.TotHrPC)
		toHrPc, _ := strconv.ParseFloat(t.TotHrPC, 64)
		totHrAprovadaPC, _ := strconv.ParseFloat(t.TotHrAprovadaPC, 64)
		fmt.Printf("Total Disponible Project: %.2f\n", totHrAprovadaPC-toHrPc)
	}

	createdAt := t.CreatedAt.Format(time.RFC3339)
	changedAt := t.ChangedAt.Format(time.RFC3339)
	if humanDates {
		createdAt = humanizeTime(t.CreatedAt)
		changedAt = humanizeTime(t.ChangedAt)
	}

	fmt.Printf("Created At: %s\n", createdAt)
	fmt.Printf("Changed At: %s\n\n", changedAt)

	utils.SectionStyle.Println("Description")
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println(t.Description)
	fmt.Println()

	if t.CreatedBy.Name != "" {
		fmt.Println("Created By:")
		fmt.Printf("%s <%s> | Phone: %s\n\n", t.CreatedBy.Name, t.CreatedBy.Email, t.CreatedBy.Phone)
	}

	if t.ProcessorDetail.Name != "" {
		utils.TitleStyle.Println("Processor:")
		utils.TitleStyle.Printf("%s <%s>\n\n", t.ProcessorDetail.Name, t.ProcessorDetail.Email)
	}

	sort.Slice(t.Texts, func(i, j int) bool {
		return t.Texts[i].TDFCreatedAt.Before(t.Texts[j].TDFCreatedAt)
	})

	fmt.Println("--- Texts ---")
	for _, tx := range t.Texts {
		tsCreated := tx.TDFCreatedAt.Format("2006-01-02 15:04")
		if humanDates {
			tsCreated = humanizeTime(tx.TDFCreatedAt)
		}
		utils.MutedStyle.Printf("\n[%s]\n", tsCreated)
		displayName := tx.TDFUser
		if tx.UserInformation != nil && tx.UserInformation.Name != "" {
			displayName = tx.UserInformation.Name
		}
		utils.TitleStyle.Printf("[%s (%s)]\n", displayName, tx.TdID)
		fmt.Println(formatTextBlock(stripHTML(tx.Text), 90))
		fmt.Println()
	}

	fmt.Printf("Created At: %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Changed At: %s\n\n", changedAt)

	if len(t.Attachments) > 0 {
		fmt.Println("--- Attachments ---")
		for _, a := range t.Attachments {
			fmt.Printf("%s by %s\n", a.FileName, a.CreatedBy.Name)
		}
	}
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var spaceRe = regexp.MustCompile(`\s{2,}`)
var blankLineRe = regexp.MustCompile(`\n{3,}`)

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = htmlTagRe.ReplaceAllString(s, "")

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		lines[i] = spaceRe.ReplaceAllString(line, " ")
	}

	out := strings.Join(lines, "\n")
	return strings.Trim(out, "\n")
}

func formatTextBlock(s string, width int) string {
	paras := strings.Split(s, "\n")
	for i, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			paras[i] = ""
			continue
		}
		paras[i] = wrap.String(p, width)
	}
	out := strings.Join(paras, "\n")
	out = blankLineRe.ReplaceAllString(out, "\n\n")
	return strings.TrimSpace(out)
}

func saveFile(filename, fileContentBase string) error {
	data, err := decodeFileContent(fileContentBase)
	if err != nil {
		return err
	}
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, bytes.NewReader(data)); err != nil {
		return err
	}
	fmt.Printf("Saved: %s\n", filename)
	return nil
}

func decodeFileContent(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")

	isHex := len(s)%2 == 0
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			isHex = false
			break
		}
	}
	if isHex {
		return hex.DecodeString(s)
	}

	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	if data, err := base64.StdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse("20060102150405", s)
	return t
}

func sortTickets(tickets []mantis.TicketResponse, by SortBy, order string) {
	parseSLA := func(s string) int {
		v, _ := strconv.Atoi(strings.TrimSuffix(s, "%"))
		return v
	}
	priorityRank := func(p string) int {
		parts := strings.Split(p, ":")
		v, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		return v
	}

	var less func(i, j int) bool
	switch by {
	case Created:
		less = func(i, j int) bool {
			return parseTime(tickets[i].TicketCreated).Before(parseTime(tickets[j].TicketCreated))
		}
	case Sla:
		less = func(i, j int) bool { return parseSLA(tickets[i].PercSLA) < parseSLA(tickets[j].PercSLA) }
	case Priority:
		less = func(i, j int) bool {
			return priorityRank(tickets[i].Priority) < priorityRank(tickets[j].Priority)
		}
	case Status:
		less = func(i, j int) bool { return tickets[i].Status < tickets[j].Status }
	case Number:
		less = func(i, j int) bool { return tickets[i].TicketNumber < tickets[j].TicketNumber }
	default:
		less = func(i, j int) bool { return true }
	}

	if order == "desc" {
		sort.Slice(tickets, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(tickets, less)
	}
}

func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func loadAndMergeCachedTickets() ([]mantis.TicketResponse, error) {
	files, err := cache.ListCacheFiles("tickets_")
	if err != nil {
		return nil, err
	}

	merged := make(map[string]mantis.TicketResponse)
	for _, name := range files {
		tickets, err := cache.ReadFromCache[mantis.TicketResponse](name)
		if err != nil {
			log.Printf("Skipping cache file %s: %v", name, err)
			continue
		}
		for _, t := range tickets {
			merged[t.TicketNumber] = t
		}
	}

	out := make([]mantis.TicketResponse, 0, len(merged))
	for _, t := range merged {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TicketNumber < out[j].TicketNumber })
	return out, nil
}

func ticketsToLines(tickets []mantis.TicketResponse) []string {
	const (
		numW  = 10
		prioW = 16
		dateW = 12
		descW = 70
	)

	lines := make([]string, 0, len(tickets))
	for _, t := range tickets {
		created := parseTime(t.TicketCreated).Format("2006-01-02")
		if humanDates {
			created = humanizeTime(parseTime(t.TicketCreated))
		}

		desc := strings.ReplaceAll(t.Description, "\n", " ")
		if len(desc) > descW {
			desc = desc[:descW-1] + "…"
		}

		lines = append(lines, fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s",
			numW, utils.SuccessStyle.Sprint(t.TicketNumber),
			prioW, colorPriority(t.Priority),
			dateW, utils.MutedStyle.Sprint(created),
			descW, desc,
		))
	}
	return lines
}

func colorPriority(p string) string {
	switch {
	case strings.HasPrefix(p, "1"):
		return utils.HighPriority.Sprint(p)
	case strings.HasPrefix(p, "2"):
		return utils.MediumPriority.Sprint(p)
	default:
		return utils.LowPriority.Sprint(p)
	}
}

func colorSLA(s string) string {
	v, _ := strconv.Atoi(strings.TrimSuffix(s, "%"))
	switch {
	case v < 50:
		return utils.SlaBad.Sprintf("%s", s)
	case v < 80:
		return utils.SlaWarn.Sprintf("%s", s)
	default:
		return utils.SlaGood.Sprintf("%s", s)
	}
}
