package analysis

import (
	"bytes"
	"strings"
	"sync"
)

var (
	StrBufPool = sync.Pool{
		New: func() interface{} {
			sb := new(strings.Builder)
			sb.Grow(16 * 1024)
			sb.Reset()
			return sb
		},
	}
	BytBufPool = sync.Pool{
		New: func() interface{} {
			buf := new(bytes.Buffer)
			buf.Grow(16 * 1024)
			buf.Reset()
			return buf
		},
	}
)
