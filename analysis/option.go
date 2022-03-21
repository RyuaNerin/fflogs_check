package analysis

import (
	"ffxiv_check/ffxiv"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RequestData struct {
	Service    string `json:"service"`
	Preset     string `json:"preset"`
	CharName   string `json:"char_name"`
	CharServer string `json:"char_server"`
	CharRegion string `json:"char_region"`
}

func (rd *RequestData) CheckOptionValidation() bool {
	rd.CharName = strings.TrimSpace(rd.CharName)
	rd.CharServer = strings.TrimSpace(rd.CharServer)
	rd.CharRegion = strings.TrimSpace(rd.CharRegion)

	lenCharName := utf8.RuneCountInString(rd.CharName)
	lenCharServer := utf8.RuneCountInString(rd.CharServer)
	lenCharRegion := utf8.RuneCountInString(rd.CharRegion)

	switch {
	case lenCharName < 2:
	case lenCharName > 20:
	case lenCharServer < 3:
	case lenCharServer > 10:
	case lenCharRegion < 2:
	case lenCharRegion > 5:
	default:
		return true
	}

	if rd.CharRegion != "kr" {
		return false
	}

	return false
}

func (rd *RequestData) Hash() uint64 {
	h := fnv.New64()

	b := make([]byte, 8)
	append := func(s string) {
		for _, c := range s {
			r := unicode.ToLower(c)

			if r >= 0 {
				if r < utf8.RuneSelf {
					b[0] = byte(r)
					h.Write(b[:1])
				} else {
					n := utf8.EncodeRune(b, r)
					h.Write(b[:n])
				}
			}
		}
	}

	var ss ffxiv.SkillSets
	if rd.CharRegion == "kr" {
		ss = ffxiv.Korea
	} else {
		ss = ffxiv.Global
	}

	fmt.Fprint(
		h,
		ss.Hash, "|",
	)
	append(rd.Service)
	append(rd.Preset)
	append(rd.CharName)
	append(rd.CharServer)
	append(rd.CharRegion)

	return h.Sum64()
}
