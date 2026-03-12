package cmd

import (
	"log"

	"github.com/Salvadego/IntraCLI/utils"
)

// resolveFilter resolves a --filter (or --filter-day) flag value.
//
// Accepted forms:
//
//	""              no-op, returns ""
//	@savedName      looks up savedName in the provided map; fatal if not found
//	any other str   treated as a raw qlvm query and returned unchanged
//
// Examples:
//
//	resolveFilter("@toyoMeetings", appConfig.SavedFilters)
//	resolveFilter(`project = toyo AND .desc = meeting`, appConfig.SavedFilters)
func resolveFilter(value string, saved map[string]string) string {
	q, err := utils.ResolveFilter(value, saved)
	if err != nil {
		log.Fatal(err)
	}
	return q
}
