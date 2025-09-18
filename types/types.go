package types

type TimesheetFilter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Type          string `yaml:"type"`
	Quantity      string `yaml:"quantity"`
	FromDate      string `yaml:"fromDate"`
	ToDate        string `yaml:"toDate"`
	Ticket        string `yaml:"ticket"`
	Project       string `yaml:"project"`
	HasTicketOnly bool   `yaml:"hasTicketOnly"`
}

var (
	TimesheetTypeLookup        = map[string]string{}
	TimesheetTypeInverseLookup = map[string]string{}
)
