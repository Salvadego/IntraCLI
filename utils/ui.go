package utils

import (
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	TitleStyle   = color.New(color.FgCyan, color.Bold)
	SectionStyle = color.New(color.FgYellow, color.Bold)
	ErrorStyle   = color.New(color.FgRed, color.Bold)
	SuccessStyle = color.New(color.FgGreen)
	MutedStyle   = color.New(color.FgHiBlack)

	HighPriority   = color.New(color.FgRed, color.Bold)
	MediumPriority = color.New(color.FgYellow)
	LowPriority    = color.New(color.FgGreen)

	SlaBad  = color.New(color.FgRed)
	SlaWarn = color.New(color.FgYellow)
	SlaGood = color.New(color.FgGreen)
)

func init() {
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		color.NoColor = true
	}
}
