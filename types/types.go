package types

type TimesheetFilter struct {
	Name          string `yaml:"name"`
	FromDate      string `yaml:"fromDate"`
	ToDate        string `yaml:"toDate"`
	Ticket        string `yaml:"ticket"`
	Project       string `yaml:"project"`
	HasTicketOnly bool   `yaml:"hasTicketOnly"`
}
