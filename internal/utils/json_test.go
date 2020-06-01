package utils_test

import (
	"reflect"
	"sort"
	"testing"

	"mkuznets.com/go/ytbackup/internal/utils"
)

var jsonData = []byte(`
{
  "a": "aa",
  "b": {
    "a": "bb",
    "c": {
      "a": "cc"
    },
    "d": [{"a": "dd"}, {"a": "aa"}, {"a": "bb"}]
  }
}
`)

func TestDecodeKeys(t *testing.T) {
	values := make([]string, 0)

	err := utils.ExtractByKey(jsonData, "a", func(v string) {
		values = append(values, v)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(values)
	expected := []string{"aa", "bb", "cc", "dd"}
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("unexpected result: %v, expected %v", values, expected)
	}
}
