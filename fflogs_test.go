package main

import (
	"context"
	"ffxiv_check/analysis"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func TestGetBuffUsage(t *testing.T) {
	opt := analysis.AnalyzeOptions{
		Context:              context.Background(),
		CharName:             "륜아린",
		CharServer:           "Moogle",
		CharRegion:           "KR",
		Zone:                 38,
		EncouterId:           77,
		AdditionalPartitions: []int{17},
	}

	r, err := analysis.Analyze(&opt)
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
