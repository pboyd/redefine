//go:build amd64

package redefine_test

import (
	"encoding/json"
	"fmt"

	"github.com/pboyd/redefine"
)

func ExampleOriginal() {
	redefine.Func(json.Marshal, func(v any) ([]byte, error) {
		// Pass strings through
		if _, ok := v.(string); ok {
			return redefine.Original(json.Marshal)(v)
		}

		return []byte(`{"nah": true}`), nil
	})
	defer redefine.Restore(json.Marshal)

	buf, _ := json.Marshal("A string")
	fmt.Println(string(buf))

	buf, _ = json.Marshal(123)
	fmt.Println(string(buf))
	// Output:
	// "A string"
	// {"nah": true}
}
