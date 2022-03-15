package perfection

import (
	"text/template"

	"ffxiv_check/share"
)

var (
	tmplEncounterRankings = template.Must(template.ParseFiles("analysis/perfection/query/tmplEncounterRankings.tmpl"))
	tmplReportSummary     = template.Must(template.ParseFiles("analysis/perfection/query/tmplReportSummary.tmpl"))
	tmplReportCastsEvents = template.Must(template.ParseFiles("analysis/perfection/query/tmplReportCastsEvents.tmpl"))

	tmplResult = template.Must(
		template.New("template.tmpl.htm").Funcs(share.TemplateFuncMap).ParseFiles("analysis/perfection/template.tmpl.htm"),
	)
)
