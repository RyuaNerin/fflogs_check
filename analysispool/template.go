package analysispool

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"sync"
)

var (
	tmplAnalysis = template.Must(
		template.New("analysis.tmpl.htm").
			Funcs(
				template.FuncMap{
					"fn": func(value interface{}) string {
						switch e := value.(type) {
						case float32:
							return fmt.Sprintf("%.2f", e)
						case float64:
							return fmt.Sprintf("%.2f", e)
						case int:
							return strconv.Itoa(e)
						}
						return ""
					},
				},
			).
			ParseFiles("analysispool/resources/analysis.tmpl.htm"),
	)

	tmplAnalysisPool = sync.Pool{
		New: func() interface{} {
			b := new(bytes.Buffer)
			b.Grow(64 * 1024)

			return b
		},
	}
)
