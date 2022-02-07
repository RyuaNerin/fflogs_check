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

type SkillData struct {
	ID         int
	Name       string
	Cooldown   int
	IconUrl    string
	OrderIndex int
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
				SkillDataEachJob[d[i]] = nil
			}

		case "1":
			if strings.HasPrefix(d[1], "#") {
				continue
			}

			id, err := strconv.Atoi(d[2])
			if err != nil {
				panic(err)
			}

			cooldown, _ := strconv.Atoi(d[3])

			SkillDataMap[id] = SkillData{
				Name:       d[1],
				ID:         id,
				Cooldown:   cooldown,
				IconUrl:    d[4],
				OrderIndex: orderIndex,
			}
			orderIndex++

			for i := 5; i < len(d); i++ {
				if d[i] != "" {
					SkillDataEachJob[columnJob[i]] = append(SkillDataEachJob[columnJob[i]], id)
				}
			}
		}
	}
}
