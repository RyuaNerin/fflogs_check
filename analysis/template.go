package analysis

import "text/template"

var (
	tmplEncounterRankings = template.Must(template.ParseFiles("analysis/query/tmplEncounterRankings.tmpl"))
	tmplReportSummary     = template.Must(template.ParseFiles("analysis/query/tmplReportSummary.tmpl"))
	tmplReportCastsEvents = template.Must(template.ParseFiles("analysis/query/tmplReportCastsEvents.tmpl"))
)
