package share

import (
	"text/template"

	"github.com/dustin/go-humanize"
)

var (
	TemplateFuncMap = template.FuncMap{
		"fn": func(value interface{}) string {
			switch e := value.(type) {
			case float32:
				return humanize.CommafWithDigits(float64(e), 1)
			case float64:
				return humanize.CommafWithDigits(e, 1)
			case int:
				return humanize.Comma(int64(e))
			}
			return ""
		},
	}
)
