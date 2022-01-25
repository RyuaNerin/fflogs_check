package ffxiv

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/dimchansky/utfbom"
)

type SkillData struct {
	ID       int
	Name     string
	Cooldown int
}

var (
	SkillDataEachJob = make(map[string][]int)
	SkillDataMap     = make(map[int]SkillData)
)

func init() {
	fs, err := os.Open("ffxiv/skills.csv")
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	sr, _ := utfbom.Skip(fs)

	var defenseBuff []SkillData

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
			if len(defenseBuff) == 0 {
				defenseBuff = make([]SkillData, len(d)-2)
			}
			for i := 2; i < len(d); i++ {
				defenseBuff[i-2].Name = d[i]
			}

		case "1":
			if len(defenseBuff) == 0 {
				defenseBuff = make([]SkillData, len(d)-2)
			}
			for i := 2; i < len(d); i++ {
				v, err := strconv.Atoi(d[i])
				if err != nil {
					panic(err)
				}
				defenseBuff[i-2].ID = v
			}

		case "2":
			if len(defenseBuff) == 0 {
				defenseBuff = make([]SkillData, len(d)-2)
			}
			for i := 2; i < len(d); i++ {
				if d[i] == "" {
					defenseBuff[i-2].Cooldown = -1
				} else {
					v, err := strconv.Atoi(d[i])
					if err != nil {
						panic(err)
					}
					defenseBuff[i-2].Cooldown = v
				}
			}

		case "3":
			arr := make([]int, 0, len(d)-2)
			for i := 2; i < len(d); i++ {
				if d[i] != "" {
					arr = append(arr, defenseBuff[i-2].ID)
				}
			}
			SkillDataEachJob[d[1]] = arr
		}
	}

	for _, buff := range defenseBuff {
		SkillDataMap[buff.ID] = buff
	}
}
