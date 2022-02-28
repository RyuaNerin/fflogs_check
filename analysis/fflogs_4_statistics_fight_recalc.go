package analysis

import (
	"math"

	"ffxiv_check/ffxiv"
)

// 최대 사용 가능한 횟수들 재 계산하는 부분...
func (inst *analysisInstance) buildReportFightRecalc() {
	set := func(fightData *analysisFight, max int, skillIds ...int) {
		var sum int
		for _, skillId := range skillIds {
			sd, ok := fightData.skillData[skillId]
			if ok {
				sum += sd.Used
			}
		}

		for _, skillId := range skillIds {
			sd, ok := fightData.skillData[skillId]
			if ok {
				sd.UsedForPercent = sum

				if max != -1 {
					sd.MaxForPercent = max
				}
			}
		}
	}

	setDefaultCharge := func(fightData *analysisFight, skillId int, defaultValue int) {
		sd, ok := fightData.skillData[skillId]
		if ok {
			sd.MaxForPercent = defaultValue + int(math.Ceil(float64(fightData.EndTime-fightData.StartTime)/1000.0/float64(inst.skillSets.Action[skillId].Cooldown)))
		}
	}

	isGlobal := inst.skillSets == &ffxiv.Global

	for _, fightData := range inst.Fights {
		if !fightData.DoneEvents || !fightData.DoneSummary {
			continue
		}

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

		case "DarkKnight":
			if isGlobal {
				setDefaultCharge(fightData, 25754, 2) // Oblation 2회
			}

		case "Gunbreaker":
			set(
				fightData,
				-1,
				16161, // 돌의 심장
				25758, // Heart of Corundum
			)

			if isGlobal {
				setDefaultCharge(fightData, 16151, 2) // 오로라 2회
			}

		case "WhiteMage":
			set(
				fightData,
				int(math.Floor(float64(fightData.EndTime-fightData.StartTime)/1000.0/30.0)),
				16531, // 위마
				16534, // 황마
			)

			if isGlobal {
				setDefaultCharge(fightData, 7430, 2)  // 실바람 2회
				setDefaultCharge(fightData, 74322, 2) // 신축 2회
			}

		case "Scholar":
			seraphimMax := fightData.skillData[16545].MaxForPercent    // 세라핌 소환
			fightData.skillData[16546].MaxForPercent = seraphimMax * 2 // 위안

		case "Astrologian":
			setDefaultCharge(fightData, 16556, 2) // 위계 2회

			// 낮별 밤별 헬리오스 합치기
			a := fightData.skillData[3601]  // 낮별
			b := fightData.skillData[17152] // 밤별

			if a != nil && b != nil {
				a.Used += b.Used
				delete(fightData.skillData, 17152)
			}

			if isGlobal {
				setDefaultCharge(fightData, 16556, 2) // 천궁의 교차 2회
				setDefaultCharge(fightData, 3590, 2)  // 점지 2회

				// 점지 3회 = Astrodyne 1회
				sd3590, ok := fightData.skillData[3590]
				if ok {
					sd25870, ok := fightData.skillData[25870] //Astrodyne
					if ok {
						sd25870.MaxForPercent = sd3590.Used / 3
					}
				}
			}

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

		case "Monk":
			setDefaultCharge(fightData, 7394, 2) // 금강의 극의 3회

		case "Summoner":
			setDefaultCharge(fightData, 25857, 2) // Magick Barrier 2회
		}
	}
}
