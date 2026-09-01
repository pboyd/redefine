//go:build amd64

package redefine_test

import (
	"encoding/json"
	"fmt"

	"github.com/pboyd/redefine"
)

//go:noinline
func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func ExampleOriginal() {
	redefine.Func(toJSON, func(v any) ([]byte, error) {
		// Pass strings through
		if _, ok := v.(string); ok {
			return redefine.Original(toJSON)(v)
		}

		return []byte(`{"nah": true}`), nil
	})
	defer redefine.Restore(toJSON)

	buf, _ := toJSON("A string")
	fmt.Println(string(buf))

	buf, _ = toJSON(123)
	fmt.Println(string(buf))
	// Output:
	// "A string"
	// {"nah": true}
}
