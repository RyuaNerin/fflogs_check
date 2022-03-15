package allstar

import (
	"encoding/json"
	"strconv"
)

type IntV struct {
	V  int
	Ok bool
}

func (fi *IntV) UnmarshalJSON(b []byte) (err error) {
	if b[0] != '"' {
		err = json.Unmarshal(b, &fi.V)
		fi.Ok = err == nil
		return err
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "-" {
		fi.Ok = true
		return nil
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	fi.V = i
	fi.Ok = true
	return nil
}

type Float32V struct {
	V  float32
	Ok bool
}

func (fi *Float32V) UnmarshalJSON(b []byte) (err error) {
	if b[0] != '"' {
		err = json.Unmarshal(b, &fi.V)
		fi.Ok = err == nil
		return err
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "-" {
		fi.Ok = true
		return nil
	}

	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return err
	}

	fi.V = float32(f)
	fi.Ok = true
	return nil
}
