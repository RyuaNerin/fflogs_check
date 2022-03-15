package allstar

import (
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
)

type preset struct {
	Name         string                   `json:"name"`
	Zone         int                      `json:"zone"`
	Difficulty   int                      `json:"difficulty"`
	Encounter    []*presetEncounter       `json:"encounter"`
	EncounterMap map[int]*presetEncounter `json:"-"`
	Partition    []*presetPartition       `json:"partition"`
	PartitionMap map[int]*presetPartition `json:"-"`
}
type presetEncounter struct {
	EncounterID int    `json:"id"`
	Name        string `json:"name"`
}
type presetPartition struct {
	Name   string `json:"name"`
	Korea  int    `json:"korea"`
	Global int    `json:"global"`
	Locked bool   `json:"locked"`
}

var (
	presetMap map[string]*preset
)

func init() {
	fs, err := os.Open("analysis/allstar/presets.json")
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&presetMap)
	if err != nil && err != io.EOF {
		panic(err)
	}

	for _, preset := range presetMap {
		preset.EncounterMap = make(map[int]*presetEncounter, len(preset.Encounter))
		for _, enc := range preset.Encounter {
			preset.EncounterMap[enc.EncounterID] = enc
		}

		preset.PartitionMap = make(map[int]*presetPartition, len(preset.Partition))
		for _, enc := range preset.Partition {
			preset.PartitionMap[enc.Korea] = enc
		}
	}
}
