package zappaclang

import (
	"fmt"
	"testing"
	"time"
)

type execTestCase struct {
	Input    string
	Expected string
}

var execTests = []execTestCase{
	{"$foo = ((0xff - b0001) // (-2 ** 2)) + -1", "62"},
	{"$foo + 1", "63"},
	{"$bar = 0xbada55", "0xbada55"},
	{"$bar - $foo", "12245527"},
}

func TestExec(t *testing.T) {
	zs := NewZappacState("")
	for _, execTest := range execTests {
		fmt.Printf("Parsing %s\n", execTest)

		start := time.Now()
		nodes, err := Parse(execTest.Input)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("%s (%s): got\n\t%+v", "TestExec", elapsed, err)
			return
		}

		fmt.Printf("Execing %+v\n", nodes)
		result, err := zs.Exec(nodes, true)
		if err != nil {
			t.Errorf("%s (%s): got\n\t%v\nexpected\n\t%v", execTest, elapsed, err, execTest.Expected)
			return
		}

		if result != execTest.Expected {
			t.Errorf("%s (%s): got\n\t%s\nexpected\n\t%s", "TestExec", elapsed, result, execTest.Expected)
			return
		}

		t.Log(execTest, fmt.Sprintf("OK in %s", elapsed))
	}
}
