package share

import (
	"strings"
	"text/template"

	"github.com/dustin/go-humanize"
	jsoniter "github.com/json-iterator/go"
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
			case []float32:
				var sb strings.Builder
				for _, v := range e {
					if sb.Len() > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(humanize.CommafWithDigits(float64(v), 1))
				}
				return sb.String()
			}
			return ""
		},
		"json": func(value interface{}) string {
			s, err := jsoniter.MarshalToString(value)
			if err != nil {
				panic(err)
			}
			return s
		},
	}
)
