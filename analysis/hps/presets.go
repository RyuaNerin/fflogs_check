package hps

import (
	"os"

	jsoniter "github.com/json-iterator/go"
)

type preset struct {
	Enc  []int `json:"enc"`
	Diff int   `json:"diff"`
	Part struct {
		Global []int `json:"global"`
		Korea  []int `json:"korea"`
	} `json:"part"`
	Version string `json:"ver"`
}

var presets map[string]preset

func init() {
	fs, err := os.Open("analysis/hps/presets.json")
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&presets)
	if err != nil {
		panic(err)
	}
}
