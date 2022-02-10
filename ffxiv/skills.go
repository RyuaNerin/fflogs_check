package ffxiv

import (
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"sort"
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
	Hash   uint32
}

type SkillData struct {
	ID       int
	Name     string
	Cooldown int
	IconUrl  string

	OrderIndex      int
	WithDowntime    bool
	ContainsInScore bool
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
	loadActions("ffxiv/resources/exd/action.exh_en.csv", 40, &Global) // BA, AO
	loadActions("ffxiv/resources/exd/action.exh_ko.csv", 39, &Korea)  // AZ, AN

	update(&Korea, SkillIdDeath, "사망", "015000-015010.png", 0)
	update(&Global, SkillIdDeath, "Death", "015000-015010.png", 0)

	update(&Korea, SkillIdPotion, "강화약", "016000-016203.png", 270)
	update(&Global, SkillIdPotion, "Medicated", "016000-016203.png", 270)

	calcVerison(&Global)
	calcVerison(&Korea)
}

func update(ss *SkillSets, id int, name string, icon string, cooldown int) {
	v := ss.Action[id]
	v.Name = name
	v.IconUrl = icon
	v.Cooldown = cooldown
	v.OrderIndex = id

	ss.Action[id] = v
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
			for i := 7; i < len(d); i++ {
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
					ID:              id,
					OrderIndex:      orderIndex,
					WithDowntime:    d[5] != "",
					ContainsInScore: d[6] != "",
				}
			}
			if d[4] != "" {
				Korea.Action[id] = SkillData{
					ID:              id,
					OrderIndex:      orderIndex,
					WithDowntime:    d[5] != "",
					ContainsInScore: d[6] != "",
				}
			}

			orderIndex++

			for i := 7; i < len(d); i++ {
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

func loadActions(path string, columnCooldown int, ss *SkillSets) {
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

		md, ok := ss.Action[index]
		if !ok {
			continue
		}

		icon, err := strconv.Atoi(d[3])
		if err != nil {
			panic(err)
		}

		cooldown, err := strconv.Atoi(d[columnCooldown])
		if err != nil {
			panic(err)
		}

		md.Name = d[1]
		md.Cooldown = cooldown / 10
		md.IconUrl = fmt.Sprintf("%06d-%06d.png", (icon/1000)*1000, icon)

		ss.Action[index] = md
	}
}

func calcVerison(ss *SkillSets) {
	h := fnv.New32a()

	jobArr := make([]string, 0, len(JobOrder))
	for job := range JobOrder {
		jobArr = append(jobArr, job)
	}
	sort.Slice(
		jobArr,
		func(i, k int) bool {
			return JobOrder[jobArr[i]] < JobOrder[jobArr[k]]
		},
	)

	for _, job := range jobArr {
		fmt.Fprintf(h, "%s\n", job)

		for _, skillId := range ss.Job[job] {
			action := ss.Action[skillId]

			fmt.Fprintf(h, "%d: %+v\n", skillId, action)
		}
	}

	ss.Hash = h.Sum32()
}
