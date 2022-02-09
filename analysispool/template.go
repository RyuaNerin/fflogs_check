package analysispool

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"
)

var (
	tmplAnalysis = template.Must(
		template.New("analysis.tmpl.htm").
			Funcs(
				template.FuncMap{
					"fn": func(value float64) string { return fmt.Sprintf("%.2f", value) },
				},
			).
			ParseFiles("./analysis.tmpl.htm"),
	)

	tmplAnalysisPool = sync.Pool{
		New: func() interface{} {
			b := new(bytes.Buffer)
			b.Grow(64 * 1024)

			return b
		},
	}
)
