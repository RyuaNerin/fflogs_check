package analysispool

import (
	"os"

	jsoniter "github.com/json-iterator/go"
)

type preset struct {
	Enc  []int `json:"enc"`
	Part []int `json:"part"`
}

var presets map[string]preset

func init() {
	fs, err := os.Open("presets.json")
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&presets)
	if err != nil {
		panic(err)
	}
}
