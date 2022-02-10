package analysis

import (
	"math"

	"ffxiv_check/ffxiv"
)

// 최대 사용 가능한 횟수들 재 계산하는 부분...
func (inst *analysisInstance) buildReportFightRecalcMaxUsing() {
	for _, fightData := range inst.Fights {
		switch fightData.Job {
		case "Paladin":
			gauge := fightData.AutoAttacks * 5

			// 효월은 100 충전해서 시작
			if inst.skillSets == &ffxiv.Global {
				gauge += 100
			}

			max := int(math.Floor(float64(gauge) / 50.0))
			fightData.skillData[3542].Max = max // 방벽
			fightData.skillData[7382].Max = max // 중재

			sd, ok := fightData.skillData[25746] // Holy Sheltron
			if ok {
				sd.Max = max
			}

		case "Scholar":
			seraphimMax := fightData.skillData[16545].Max    // 세라핌 소환
			fightData.skillData[16546].Max = seraphimMax * 2 // 위안

		case "DarkKnight":
			sd, ok := fightData.skillData[25754] // Oblation
			if ok {
				sd.Max += 2 // 기본 충전량 2개
			}

		}
	}
}
