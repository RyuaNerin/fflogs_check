package analysis

import (
	"math"

	"ffxiv_check/ffxiv"
)

// 각 전투별 통계내는 부분
func (inst *analysisInstance) buildReportFight() {
	for _, fight := range inst.Fights {
		if !fight.DoneEvents {
			continue
		}

		fightTime := fight.EndTime - fight.StartTime

		for _, skillId := range inst.skillSets.Job[fight.Job] {
			skillInfo := inst.skillSets.Action[skillId]

			switch skillId {
			case ffxiv.SkillIdReduceDamangeDebuff:
				fight.skillData[skillId] = &analysisFightSkill{
					Used:           fight.Debuff.ReduceDamange.count,
					UsedForPercent: fightTime - fight.Debuff.ReduceDamange.uptime,
					MaxForPercent:  fightTime,
				}

			default:
				used := 0
				nextCooldown := 0

				switch skillId {
				case ffxiv.SkillIdDeath:
					used = len(fight.Deaths)

				case ffxiv.SkillIdPotion:
					for _, event := range fight.Buffs {
						if event.removed {
							event.timestamp = event.timestamp - ffxiv.PotionBuffTime
						}
						if nextCooldown > 0 && event.timestamp < nextCooldown {
							// 적용 후 꺼진 버프
							// 탕약 버프가 두번 뜨는 경우가 있음
							continue
						}

						used++
						nextCooldown = event.timestamp + skillInfo.Cooldown*1000
					}

				default:
					for _, event := range fight.Casts {
						if skillId != 0 && event.gameID != skillId {
							continue
						}

						used++
					}
				}

				if skillInfo.WithDowntime {
					fight.skillData[skillId] = &analysisFightSkill{
						Used:           used,
						UsedForPercent: used,
						MaxForPercent:  int(math.Ceil(float64(fightTime) / float64(skillInfo.Cooldown*1000))),
					}
				} else {
					fight.skillData[skillId] = &analysisFightSkill{
						Used:           used,
						UsedForPercent: used,
						MaxForPercent:  0,
					}
				}
			}
		}
	}

	share := func(fight *analysisFight, skillIds ...int) {
		arr := make([]*analysisFightSkill, len(skillIds))

		var sum int
		for i, skillId := range skillIds {
			sd, ok := fight.skillData[skillId]
			if ok {
				arr[i] = sd
				sum += arr[i].Used
			}
		}

		for _, v := range arr {
			if v == nil {
				continue
			}

			v.UsedForPercent = sum
		}
	}

	// 쿨을 공유하는 스킬들 하나로 묶기
	for _, fflogsFight := range inst.Fights {
		switch fflogsFight.Job {
		case "Paladin":
			share(
				fflogsFight,
				3542,  // 방벽
				7382,  // 중재
				25746, // Holy Sheltron
			)

		case "Warrior":
			share(
				fflogsFight,
				3551,  // 직감
				16464, // 분노
				25751, // Bloodwhetting
			)

		case "Gunbreaker":
			share(
				fflogsFight,
				16161, // 돌의 심장
				25758, // Heart of Corundum
			)
		}
	}
}
