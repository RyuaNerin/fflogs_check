package ffxiv

import (
	"encoding/csv"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dimchansky/utfbom"
)

const (
	SkillIdDeath               = -1
	SkillIdPotion              = -2
	SkillIdReduceDamangeDebuff = -3

	PotionBuffTime = 30 * 1000
)

type GameData struct {
	Version string
	Job     map[string][]int
	Action  map[int]SkillData
	Hash    uint32

	h hash.Hash32
}

type SkillData struct {
	ID       int
	Name     string
	Cooldown int
	IconUrl  string

	OrderIndex      int
	WithDowntime    bool
	ContainsInScore bool

	Level int
}

var (
	GameDataMap = make(map[string]*GameData)
)

func init() {
	GameDataMap["54"] = load("54", "ffxiv/resources/skills_54.csv", "ffxiv/resources/exd/skills_54.csv")
	GameDataMap["60"] = load("60", "ffxiv/resources/skills_60.csv", "ffxiv/resources/exd/skills_60.csv")
}

func load(version, csv, exd string) *GameData {
	gd := &GameData{
		Job:     make(map[string][]int),
		Action:  make(map[int]SkillData),
		Version: version,
		h:       fnv.New32(),
	}

	gd.loadCsv(csv)
	gd.loadExd(exd, 40) // AO

	gd.update(SkillIdDeath, "사망", "015000-015010.png", 0)
	gd.update(SkillIdPotion, "강화약", "016000-016203.png", 270)
	gd.update(SkillIdReduceDamangeDebuff, "주는 피해량 감소", "015000-015520.png", 0)

	gd.Hash = gd.h.Sum32()

	return gd
}

func (gd *GameData) update(id int, name string, icon string, cooldown int) {
	v := gd.Action[id]
	v.Name = name
	v.IconUrl = icon
	v.Cooldown = cooldown

	gd.Action[id] = v
}

func (gd *GameData) loadCsv(path string) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	_, err = io.Copy(gd.h, fs)
	if err != nil && err != io.EOF {
		panic(err)
	}

	_, err = fs.Seek(0, os.SEEK_SET)
	if err != nil {
		panic(err)
	}

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
				gd.Job[d[i]] = nil
			}

		case "1":
			if strings.HasPrefix(d[1], "#") {
				continue
			}

			id, err := strconv.Atoi(d[2])
			if err != nil {
				panic(err)
			}

			gd.Action[id] = SkillData{
				ID:              id,
				OrderIndex:      orderIndex,
				WithDowntime:    d[3] != "",
				ContainsInScore: d[4] != "",
			}

			orderIndex++

			for i := 5; i < len(d); i++ {
				if d[i] != "" {
					gd.Job[columnJob[i]] = append(gd.Job[columnJob[i]], id)
				}
			}
		}
	}
}

func (gd *GameData) loadExd(path string, columnCooldown int) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	_, err = io.Copy(gd.h, fs)
	if err != nil && err != io.EOF {
		panic(err)
	}

	_, err = fs.Seek(0, os.SEEK_SET)
	if err != nil {
		panic(err)
	}

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

		md, ok := gd.Action[index]
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

		gd.Action[index] = md
	}
}
