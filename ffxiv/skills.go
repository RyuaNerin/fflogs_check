package ffxiv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dimchansky/utfbom"
)

const (
	SkillIdDeath  = -1
	SkillIdPotion = -2

	PotionBuffTime = 30 * 1000
)

type SkillSets struct {
	Job    map[string][]int
	Action map[int]SkillData
}

type SkillData struct {
	ID         int
	Name       string
	Cooldown   int
	IconUrl    string
	OrderIndex int
}

var (
	Korea = SkillSets{
		Job:    make(map[string][]int),
		Action: make(map[int]SkillData),
	}
	Global = SkillSets{
		Job:    make(map[string][]int),
		Action: make(map[int]SkillData),
	}
)

func init() {
	loadSkills("ffxiv/resources/skills.csv")
	loadActions("ffxiv/resources/exd/action.exh_en.csv", 37, 40, &Global) // AL, AO
	loadActions("ffxiv/resources/exd/action.exh_ko.csv", 37, 39, &Korea)  // AL, AN

	Korea.Action[SkillIdDeath] = SkillData{
		ID:         SkillIdDeath,
		Name:       "사망",
		IconUrl:    "015000-015010.png",
		OrderIndex: SkillIdDeath,
	}
	Global.Action[SkillIdDeath] = SkillData{
		ID:         SkillIdDeath,
		Name:       "Death",
		IconUrl:    "015000-015010.png",
		OrderIndex: SkillIdDeath,
	}

	Korea.Action[SkillIdPotion] = SkillData{
		ID:         SkillIdPotion,
		Name:       "강화약",
		IconUrl:    "016000-016203.png",
		OrderIndex: SkillIdPotion,
		Cooldown:   270,
	}
	Global.Action[SkillIdPotion] = SkillData{
		ID:         SkillIdPotion,
		Name:       "Medicated",
		IconUrl:    "016000-016203.png",
		OrderIndex: SkillIdPotion,
		Cooldown:   270,
	}

}

func loadSkills(path string) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	sr, _ := utfbom.Skip(fs)

	var columnJob []string

	orderIndex := 0

	cr := csv.NewReader(sr)
	for {
		d, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		switch d[0] {
		case "0":
			columnJob = make([]string, len(d))
			for i := range d {
				columnJob[i] = fmt.Sprint(d[i])
			}
			for i := 5; i < len(d); i++ {
				Korea.Job[d[i]] = nil
				Global.Job[d[i]] = nil
			}

		case "1":
			if strings.HasPrefix(d[1], "#") {
				continue
			}

			id, err := strconv.Atoi(d[2])
			if err != nil {
				panic(err)
			}

			if d[3] != "" {
				Global.Action[id] = SkillData{
					ID:         id,
					OrderIndex: orderIndex,
				}
			}
			if d[4] != "" {
				Korea.Action[id] = SkillData{
					ID:         id,
					OrderIndex: orderIndex,
				}
			}

			orderIndex++

			for i := 5; i < len(d); i++ {
				if d[i] != "" {
					if d[3] != "" {
						Global.Job[columnJob[i]] = append(Global.Job[columnJob[i]], id)
					}
					if d[4] != "" {
						Korea.Job[columnJob[i]] = append(Korea.Job[columnJob[i]], id)
					}
				}
			}
		}
	}
}

func loadActions(path string, columnIsAbility int, columnCooldown int, ss *SkillSets) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	sr, _ := utfbom.Skip(fs)

	cr := csv.NewReader(sr)
	for {
		d, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if len(d) < columnCooldown {
			continue
		}

		index, err := strconv.Atoi(d[0])
		if err != nil {
			continue
		}

		icon, err := strconv.Atoi(d[3])
		if err != nil {
			continue
		}

		var cooldown int

		if d[columnIsAbility] == "true" {
			cooldown, err = strconv.Atoi(d[columnCooldown])
			if err != nil {
				continue
			}

			if cooldown < 50 {
				cooldown = 0
			}
		}

		md, ok := ss.Action[index]
		if ok {
			md.Name = d[1]
			md.Cooldown = cooldown / 10
			md.IconUrl = fmt.Sprintf("%06d-%06d.png", (icon/1000)*1000, icon)

			ss.Action[index] = md
		}

	}
}
