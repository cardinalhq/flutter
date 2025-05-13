package config

import (
	"encoding/json"
	"io"
)

func JSONDecode(j io.Reader, target any) error {
	decoder := json.NewDecoder(j)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
