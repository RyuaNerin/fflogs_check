package allstar

import (
	"sort"

	"ffxiv_check/ffxiv"
)

func (inst *analysisInstance) UpdateMapToSlice() {
	inst.tmplData.Jobs = make([]*tmplDataJob, 0, len(inst.tmplData.jobsMap))
	for _, jobData := range inst.tmplData.jobsMap {
		inst.tmplData.Jobs = append(inst.tmplData.Jobs, jobData)

		jobData.Partitions = make([]*tmplDataPartition, 0, len(jobData.partitionsMap))
		for _, partData := range jobData.partitionsMap {
			jobData.Partitions = append(jobData.Partitions, partData)

			partData.Encounters = make([]*tmplDataEncounter, 0, len(partData.encountersMap))
			for _, encData := range partData.encountersMap {
				partData.Encounters = append(partData.Encounters, encData)
			}
		}
	}

	sort.Slice(
		inst.tmplData.Jobs,
		func(i, k int) bool {
			return ffxiv.JobOrder[inst.tmplData.Jobs[i].Job] < ffxiv.JobOrder[inst.tmplData.Jobs[k].Job]
		},
	)
	for _, jobData := range inst.tmplData.Jobs {
		sort.Slice(
			jobData.Partitions,
			func(i, k int) bool {
				ir, kr := -1, -1

				for ii, v := range inst.Preset.Partition {
					if v.Korea == jobData.Partitions[i].PartitionIDKorea {
						ir = ii
					}
				}
				for ii, v := range inst.Preset.Partition {
					if v.Korea == jobData.Partitions[k].PartitionIDKorea {
						kr = ii
					}
				}

				return ir < kr
			},
		)

		for _, partData := range jobData.Partitions {
			sort.Slice(
				partData.Encounters,
				func(i, k int) bool {
					ir, kr := -1, -1

					for ii, v := range inst.Preset.Encounter {
						if v.EncounterID == partData.Encounters[i].EncounterID {
							ir = ii
						}
					}
					for ii, v := range inst.Preset.Encounter {
						if v.EncounterID == partData.Encounters[k].EncounterID {
							kr = ii
						}
					}

					return ir < kr
				},
			)
		}
	}
}
