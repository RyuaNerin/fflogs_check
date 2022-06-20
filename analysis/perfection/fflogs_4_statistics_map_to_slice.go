package perfection

import (
	"sort"

	"ffxiv_check/ffxiv"
)

// Map 에 저장된걸 배열로 옮기면서 재정렬하는 부분.
func (inst *analysisInstance) buildReportMapToSlice() {
	inst.stat.Jobs = make([]*statisticJob, 0, len(inst.stat.jobsMap))
	for _, jobData := range inst.stat.jobsMap {
		inst.stat.Jobs = append(inst.stat.Jobs, jobData)
	}
	sort.Slice(
		inst.stat.Jobs,
		func(i, k int) bool {
			return ffxiv.JobOrder[inst.stat.Jobs[i].Job] < ffxiv.JobOrder[inst.stat.Jobs[k].Job]
		},
	)

	inst.stat.Encounters = make([]*statisticEncounter, 0, len(inst.stat.encountersMap))
	for _, encData := range inst.stat.encountersMap {
		inst.stat.Encounters = append(inst.stat.Encounters, encData)

		encData.Jobs = make([]*statisticEncounterJob, 0, len(encData.jobsMap))
		for _, encJobData := range encData.jobsMap {
			encData.Jobs = append(encData.Jobs, encJobData)

			encJobData.Skills = make([]*statisticSkill, 0, len(encJobData.skillsMap))
			for _, encJobSkillData := range encJobData.skillsMap {
				encJobData.Skills = append(encJobData.Skills, encJobSkillData)
			}
			sort.Slice(
				encJobData.Skills,
				func(i, k int) bool {
					return inst.gameData.Action[encJobData.Skills[i].Info.ID].OrderIndex < inst.gameData.Action[encJobData.Skills[k].Info.ID].OrderIndex
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
		inst.stat.Encounters,
		func(i, k int) bool {
			ir, kr := -1, -1

			for ii, v := range inst.InpEncounterIDs {
				if v == inst.stat.Encounters[i].ID {
					ir = ii
				}
			}
			for ii, v := range inst.InpEncounterIDs {
				if v == inst.stat.Encounters[k].ID {
					kr = ii
				}
			}

			return ir < kr
		},
	)
}
