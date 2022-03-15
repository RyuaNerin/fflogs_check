package allstar

import (
	"sort"

	"ffxiv_check/ffxiv"
)

func (inst *analysisInstance) UpdateMapToSlice() {
	inst.tmplData.Partitions = make([]*tmplDataPartition, 0, len(inst.tmplData.partitionsMap))
	for _, partData := range inst.tmplData.partitionsMap {
		inst.tmplData.Partitions = append(inst.tmplData.Partitions, partData)

		partData.Jobs = make([]*tmplDataJob, 0, len(partData.jobsMap))
		for _, jobData := range partData.jobsMap {
			partData.Jobs = append(partData.Jobs, jobData)

			jobData.Encounters = make([]*tmplDataEncounter, 0, len(jobData.encountersMap))
			for _, encData := range jobData.encountersMap {
				jobData.Encounters = append(jobData.Encounters, encData)
			}
		}
	}

	sort.Slice(
		inst.tmplData.Partitions,
		func(i, k int) bool {
			ir, kr := -1, -1

			for ii, v := range inst.Preset.Partition {
				if v.Korea == inst.tmplData.Partitions[i].PartitionIDKorea {
					ir = ii
				}
			}
			for ii, v := range inst.Preset.Partition {
				if v.Korea == inst.tmplData.Partitions[k].PartitionIDKorea {
					kr = ii
				}
			}

			return ir < kr
		},
	)
	for _, partData := range inst.tmplData.Partitions {
		sort.Slice(
			partData.Jobs,
			func(i, k int) bool {
				return ffxiv.JobOrder[partData.Jobs[i].Job] < ffxiv.JobOrder[partData.Jobs[k].Job]
			},
		)
		for _, jobData := range partData.Jobs {
			sort.Slice(
				jobData.Encounters,
				func(i, k int) bool {
					ir, kr := -1, -1

					for ii, v := range inst.Preset.Encounter {
						if v.EncounterID == jobData.Encounters[i].EncounterID {
							ir = ii
						}
					}
					for ii, v := range inst.Preset.Encounter {
						if v.EncounterID == jobData.Encounters[k].EncounterID {
							kr = ii
						}
					}

					return ir < kr
				},
			)
		}
	}
}
