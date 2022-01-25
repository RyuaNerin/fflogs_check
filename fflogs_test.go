package main

import (
	"strings"
	"testing"

	"ffxiv_check/fflogs"

	jsoniter "github.com/json-iterator/go"
)

func TestGetBuffUsage(t *testing.T) {
	r, err := fflogs.GetBuffUsage("륜아린", "Moogle", 74, true)
	if err != nil {
		panic(err)
	}

	var sb strings.Builder
	je := jsoniter.NewEncoder(&sb)

	je.SetIndent("", "    ")
	err = je.Encode(r)
	if err != nil {
		panic(err)
	}

	print(sb.String())
}
