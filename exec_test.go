package zappaclang

import (
	"fmt"
	"testing"
	"time"
)

var execTests = map[string]string{
	"$foo = ((0xff - b0001) // (-2 ** 2)) + -1": "62",
	"$foo + 1":        "63",
	"$bar = 0xbada55": "0xbada55",
	"$bar - $foo":     "12245527",
}

func TestExec(t *testing.T) {
	zs := NewZappacState("")
	for input, expected := range execTests {
		fmt.Printf("Parsing %s\n", input)

		start := time.Now()
		nodes, err := Parse(input)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("%s (%s): got\n\t%+v", "TestExec", elapsed, err)
			return
		}

		fmt.Printf("Execing %+v\n", nodes)
		result, err := zs.Exec(nodes)
		if err != nil {
			t.Errorf("%s (%s): got\n\t%v\nexpected\n\t%v", input, elapsed, err, expected)
			return
		}

		if result != expected {
			t.Errorf("%s (%s): got\n\t%s\nexpected\n\t%s", "TestExec", elapsed, result, expected)
			return
		}

		t.Log(input, fmt.Sprintf("OK in %s", elapsed))
	}
}
