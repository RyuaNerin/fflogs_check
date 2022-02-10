package analysis

import (
	"sort"

	"ffxiv_check/ffxiv"
)

// Map 에 저장된걸 배열로 옮기면서 재정렬하는 부분.
func (inst *analysisInstance) buildReportMapToSlice(stat *Statistic) {
	stat.Jobs = make([]*StatisticJob, 0, len(stat.jobsMap))
	for _, jobData := range stat.jobsMap {
		stat.Jobs = append(stat.Jobs, jobData)
	}
	sort.Slice(
		stat.Jobs,
		func(i, k int) bool {
			return ffxiv.JobOrder[stat.Jobs[i].Job] < ffxiv.JobOrder[stat.Jobs[k].Job]
		},
	)

	stat.Encounters = make([]*StatisticEncounter, 0, len(stat.encountersMap))
	for _, encData := range stat.encountersMap {
		stat.Encounters = append(stat.Encounters, encData)

		encData.Jobs = make([]*StatisticEncounterJob, 0, len(encData.jobsMap))
		for _, encJobData := range encData.jobsMap {
			encData.Jobs = append(encData.Jobs, encJobData)

			encJobData.Skills = make([]*StatisticSkill, 0, len(encJobData.skillsMap))
			for _, encJobSkillData := range encJobData.skillsMap {
				encJobData.Skills = append(encJobData.Skills, encJobSkillData)
			}
			sort.Slice(
				encJobData.Skills,
				func(i, k int) bool {
					return inst.skillSets.Action[encJobData.Skills[i].Info.ID].OrderIndex < inst.skillSets.Action[encJobData.Skills[k].Info.ID].OrderIndex
				},
			)
		}
		sort.Slice(
			encData.Jobs,
			func(i, k int) bool {
				return ffxiv.JobOrder[encData.Jobs[i].Job] < ffxiv.JobOrder[encData.Jobs[k].Job]
			},
		)
	}
	sort.Slice(
		stat.Encounters,
		func(i, k int) bool {
			ir, kr := -1, -1

			for ii, v := range inst.InpEncounterIDs {
				if v == stat.Encounters[i].ID {
					ir = ii
				}
			}
			for ii, v := range inst.InpEncounterIDs {
				if v == stat.Encounters[k].ID {
					kr = ii
				}
			}

			return ir < kr
		},
	)
}
