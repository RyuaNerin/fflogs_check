package fflogs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
)

func (rd *reportDataInner) pathReportDetailPath() string {
	return fmt.Sprintf("cache/rd_%s_%d.json", rd.Report.Code, rd.Report.FightID)
}

func (rd *reportDataInner) saveReportSummary(r FFLogsReportResponse) {
	path := rd.pathReportDetailPath()
	os.MkdirAll(filepath.Dir(path), 0700)

	fs, err := os.Create(path)
	if err != nil {
		return
	}
	defer fs.Close()

	err = jsoniter.NewEncoder(fs).Encode(&r)
	if err != nil {
		return
	}
}

func (rd *reportDataInner) loadReportSummary() (ok bool, r FFLogsReportResponse) {
	fs, err := os.Open(rd.pathReportDetailPath())
	if err != nil {
		return
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&r)
	if err != nil && err != io.EOF {
		return
	}

	ok = true
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func (rd *reportDataInner) pathReportCastsEventsPath() string {
	return fmt.Sprintf("cache/rce_%s_%d_%d_%d.json", rd.Report.Code, rd.Report.FightID, rd.SourceId, rd.EventsNextPage)
}

func (rd *reportDataInner) saveReportCastsEvents(r FFLogsReportCastsEventData) {
	path := rd.pathReportCastsEventsPath()
	os.MkdirAll(filepath.Dir(path), 0700)

	fs, err := os.Create(path)
	if err != nil {
		return
	}
	defer fs.Close()

	err = jsoniter.NewEncoder(fs).Encode(&r)
	if err != nil {
		return
	}
}

func (rd *reportDataInner) loadReportCastsEvents() (ok bool, r FFLogsReportCastsEventData) {
	fs, err := os.Open(rd.pathReportCastsEventsPath())
	if err != nil {
		return
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&r)
	if err != nil && err != io.EOF {
		return
	}

	ok = true
	return
}
