package perfection

import (
	"sort"
)

// Score, Med, Avg 등을 계산하는 부분
func (inst *analysisInstance) buildReportCalcStat() {
	for _, jobData := range inst.stat.jobsMap {
		if jobData.scoreCount > 0 {
			jobData.Score = float32(jobData.scoreSum / float64(jobData.scoreCount))
		}
	}

	for _, encData := range inst.stat.encountersMap {
		if encData.scoreCount > 0 {
			encData.Score = float32(encData.scoreSum / float64(encData.scoreCount))
		}

		for _, endJobData := range encData.jobsMap {
			if endJobData.scoreCount > 0 {
				endJobData.Score = float32(endJobData.scoreSum / float64(endJobData.scoreCount))
			}

			for _, encJobSkillData := range endJobData.skillsMap {
				sort.Slice(encJobSkillData.Usage.data, func(i, k int) bool { return encJobSkillData.Usage.data[i] < encJobSkillData.Usage.data[k] })
				sort.Slice(encJobSkillData.Cooldown.data, func(i, k int) bool { return encJobSkillData.Cooldown.data[i] < encJobSkillData.Cooldown.data[k] })

				if len(encJobSkillData.Usage.data) > 0 {
					var usageSum int = 0
					for _, u := range encJobSkillData.Usage.data {
						usageSum += u
					}
					encJobSkillData.Usage.Med = encJobSkillData.Usage.data[len(encJobSkillData.Usage.data)/2]
					encJobSkillData.Usage.Avg = float32(usageSum) / float32(len(encJobSkillData.Usage.data))
				}

				if len(encJobSkillData.Cooldown.data) > 0 {
					var cooldownSum float32 = 0
					for _, u := range encJobSkillData.Cooldown.data {
						cooldownSum += u
					}
					encJobSkillData.Cooldown.Med = encJobSkillData.Cooldown.data[len(encJobSkillData.Cooldown.data)/2]
					encJobSkillData.Cooldown.Avg = cooldownSum / float32(len(encJobSkillData.Cooldown.data))
				}
			}
		}
	}
}
