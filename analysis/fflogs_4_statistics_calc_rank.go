package analysis

import "sort"

func (inst *analysisInstance) buildReportCalcRank(stat *Statistic) {
	order := func(data []fflogsRankData) bool {
		if len(data) == 0 {
			return false
		}

		sort.Slice(
			data,
			func(aidx, bidx int) bool {
				a := data[aidx]
				b := data[bidx]

				if int(a.Rank) == int(b.Rank) {
					return a.Amount > b.Amount
				} else {
					return a.Rank > b.Rank
				}
			},
		)

		return true
	}

	for encId, dataEnc := range inst.encounterRanks {
		// Dps
		for job, dataEncJob := range dataEnc.Dps {
			d := stat.encountersMap[encId].jobsMap[job]

			if order(dataEncJob.Data) {
				d.Rank.Dps = dataEncJob.Data[0].Rank
			} else {
				d.Rank.Dps = -1
			}
		}

		// Hps
		for job, dataEncJob := range dataEnc.Hps {
			d := stat.encountersMap[encId].jobsMap[job]

			if order(dataEncJob.Data) {
				d.Rank.Hps = dataEncJob.Data[len(dataEncJob.Data)/2].Rank
			} else {
				d.Rank.Hps = -1
			}
		}
	}
}
