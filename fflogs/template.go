package fflogs

import "text/template"

var (
	tmplEncounterRankings = template.Must(template.ParseFiles("fflogs/query/tmplEncounterRankings.tmpl"))
	tmplReportSummary     = template.Must(template.ParseFiles("fflogs/query/tmplReportSummary.tmpl"))
	tmplReportCastsEvents = template.Must(template.ParseFiles("fflogs/query/tmplReportCastsEvents.tmpl"))
)
