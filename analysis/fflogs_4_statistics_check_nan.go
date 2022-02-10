package analysis

import (
	"fmt"
	"math"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

// NaN 체크는는 함수
func (inst *analysisInstance) buildReportCheckNaN(stat *Statistic) {
	var msgOnce sync.Once
	check := func(v float32) {
		if math.IsNaN(float64(v)) {
			msgOnce.Do(func() {
				err := errors.Errorf(
					"NaN : %s@%s (%s)\nEnc: %+v\nPartition: %+v\nJobs: %+v",
					inst.InpCharName, inst.InpCharServer, inst.InpCharRegion,
					inst.InpEncounterIDs,
					inst.InpAdditionalPartition,
					inst.InpCharJobs,
				)

				fmt.Printf("%+v\n", errors.WithStack(err))
				sentry.CaptureException(err)
			})
		}
	}
	for _, jobData := range stat.jobsMap {
		check(jobData.Score)
	}
	for _, encData := range stat.encountersMap {
		check(encData.Score)

		for _, encJobData := range encData.jobsMap {
			check(encJobData.Score)

			for _, encJobSkillData := range encJobData.skillsMap {
				check(encJobSkillData.Usage.Avg)
				check(encJobSkillData.Cooldown.Avg)
				check(encJobSkillData.Cooldown.Med)
			}
		}
	}
}
