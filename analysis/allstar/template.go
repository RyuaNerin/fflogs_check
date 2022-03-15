package allstar

import (
	"text/template"

	"ffxiv_check/share"
)

var (
	tmplEncounterRankings = template.Must(template.ParseFiles("analysis/allstar/query/tmplEncounterRankings.tmpl"))
	tmplEncounterRank     = template.Must(template.ParseFiles("analysis/allstar/query/tmplEncounterRank.tmpl"))

	tmplResult = template.Must(
		template.New("template.tmpl.htm").Funcs(share.TemplateFuncMap).ParseFiles("analysis/allstar/template.tmpl.htm"),
	)
)
