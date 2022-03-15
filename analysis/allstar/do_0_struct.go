package allstar

import (
	"context"
	"fmt"
)

const (
	maxRetries = 3
	workers    = 8

	maxAllstar = 10
)

type analysisInstance struct {
	ctx context.Context

	CharName   string
	CharServer string
	Jobs       map[string]bool

	tmplData *tmplData
	Preset   *preset

	progressString chan string
}

func (inst *analysisInstance) progress(format string, args ...interface{}) {
	select {
	case <-inst.ctx.Done():
	case inst.progressString <- fmt.Sprintf(format, args...):
	}
}
