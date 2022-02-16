package analysis

import (
	"math"

	"ffxiv_check/ffxiv"
)

// 최대 사용 가능한 횟수들 재 계산하는 부분...
func (inst *analysisInstance) buildReportFightRecalcMaxUsing() {
	set := func(fight *analysisFight, max int, skillIds ...int) {
		var sum int
		for _, skillId := range skillIds {
			sd, ok := fight.skillData[skillId]
			if ok {
				sum += sd.Used
			}
		}

		for _, skillId := range skillIds {
			sd, ok := fight.skillData[skillId]
			if ok {
				sd.UsedForPercent = sum

				if max != -1 {
					sd.MaxForPercent = max
				}
			}
		}
	}

	for _, fightData := range inst.Fights {
		switch fightData.Job {
		case "Paladin":
			gauge := fightData.AutoAttacks * 5

			// 효월은 100 충전해서 시작
			if inst.skillSets == &ffxiv.Global {
				gauge += 100
			}

			set(
				fightData,
				int(math.Floor(float64(gauge)/50.0)),
				3542,  // 방벽
				7382,  // 중재
				25746, // Holy Sheltron
			)

		case "Warrior":
			set(
				fightData,
				-1,
				3551,  // 직감
				16464, // 분노
				25751, // Bloodwhetting
			)

		case "Gunbreaker":
			set(
				fightData,
				-1,
				16161, // 돌의 심장
				25758, // Heart of Corundum
			)

		case "DarkKnight":
			sd, ok := fightData.skillData[25754] // Oblation
			if ok {
				sd.MaxForPercent += 2 // 기본 충전량 2개
			}

		case "WhiteMage":
			set(
				fightData,
				int(math.Floor(float64(fightData.EndTime-fightData.StartTime)/1000.0/30.0)),
				16531, // 위마
				16534, // 황마
			)

		case "Scholar":
			seraphimMax := fightData.skillData[16545].MaxForPercent    // 세라핌 소환
			fightData.skillData[16546].MaxForPercent = seraphimMax * 2 // 위안

		case "Sage":
			max := 3 + int(math.Floor(float64(fightData.EndTime-fightData.StartTime)/1000.0/20.0))

			sd, ok := fightData.skillData[24309] // Rizomata
			if ok {
				max += sd.Used
			}

			set(
				fightData,
				max,
				24296, // Druochole
				24303, // Taurochole
				24299, // Ixochole
				24298, // Kerachole
			)
		}
	}
}
