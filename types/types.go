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
	Negate        bool   `yaml:"negate"`
}

type DailyFilter struct {
	FromDate      string  `yaml:"fromDate"`
	ToDate        string  `yaml:"toDate"`
	MinDailyHours float64 `yaml:"minDailyHours"`
	Negate        bool    `yaml:"negate"`
	Project       string  `yaml:"project"`
	User          string  `yaml:"user"`
	HasTicketOnly bool    `yaml:"hasTicketOnly"`
	Status        string  `yaml:"status"`
}

var (
	TimesheetTypeLookup        = map[string]string{}
	TimesheetTypeInverseLookup = map[string]string{}
)
