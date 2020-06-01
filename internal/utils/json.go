package utils

import (
	"bytes"
	"encoding/json"
	"io"
)

func ExtractByKey(data []byte, key string, fun func(string)) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))

	seen := make(map[string]struct{})
	foundKey := false

	for {
		t, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if v, ok := t.(string); ok {
			if v == key {
				foundKey = true
				continue
			}
			if foundKey {
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					fun(v)
				}
			}
		}
		foundKey = false
	}

	return nil
}
