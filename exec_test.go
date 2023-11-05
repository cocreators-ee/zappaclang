package zappaclang

import (
	"fmt"
	"os"
	"testing"
	"time"
)

type execTestCase struct {
	Input    string
	Expected string
}

var execTests = []execTestCase{
	// General high level testing of parsing and logic
	{"$foo = ((0xff - b0001) // (-2 ** 2)) + -1", "0x3e"},
	{"$foo + 1", "63"},
	{"$bar = 0xbada55", "0xbada55"},
	{"$bar - $foo", "12245527"},
	{"save(foobar)", "Saved foobar"},
	{"load(foobar)", "Loaded foobar"},

	// Testing that it doesn't mangle things
	{"0xff", "0xff"},
	{"0755", "0755"},
	{"0.1243871635897613587671", "0.1243871635897613587671"},
	{"1243871635897613587671", "1243871635897613587671"},
	{"-12438716358976137671", "-12438716358976137671"},

	// Basic arithmetic tests
	{"0.1 + 0.2", "0.3"}, // Precision is off
	{"0.1 + -0.2", "-0.1"},
	{"0.1 - 0.2", "-0.1"},
	{"0.1 - 0.2", "-0.1"},
	{"0.01 + 0.02", "0.03"},
	{"1.2 + 3.4", "4.6"},
	{"1.2 + 3", "4.2"},
	{"1.2 + 0x0f", "16.2"},
	{"1.2 + b1000", "9.2"},
	{"1 + 2", "3"},
	{"1 + -2", "-1"},
	{"1 - 2", "-1"},
	{"-1 - 2", "-3"},
	{"-1 - -2", "1"},
	{"1 * 0", "0"}, // Unexpected end of input
	{"100 * 10", "1000"},
	{"100 * 1.234", "123.4"},
	{"100 * 0.00123", "0.123"},
	{"10 ** 2", "100"},
	{"(1+2)*((3-4)*5)", "-15"},
	{"5%2", "1"},
	{"6%2", "0"},
	{"1 << 10", "1024"},
	{"1024 >> 2", "256"},
	{"1024 | 8", "1032"}, // Wrong?
	{"151451 ^ 2", "151449"},
	{"151451 & 2", "2"}, // Wrong?
	{"151451 & 4", "0"}, // Wrong?
	{"~1024", "-1025"},  // Crash - not implemented
	{"10 // 3", "3"},
	{"abs(10)", "10"},  // error
	{"abs(-10)", "10"}, // error

	// Some precision loss after this, which is fine for now
	{"1 / 10000000000000000000000", "0.0000000000000000000001"},

	// Conversions
	{"dec(0xff)", "255"},
	{"dec(0755)", "493"},
	{"bin(2)", "b10"},
	{"hex(255)", "0xff"},
	{"oct(8)", "010"},

	// Errors shouldn't crash but give a decent message
	{"error", "unexpected error at position 0"},
	{"abs()", "unexpected ) at pos 4, should be following numbers, variables, or other )s"},
	{"$fo", "unknown variable $fo"},
	{"+", "unexpected + at position 0"},

	// Non-errors
	{"", ""},
	{" ", ""},
	{"            ", ""},
	{"      1     ", "1"},
}

func TestExec(t *testing.T) {
	dir, err := os.MkdirTemp("", "zappac-test")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	StoragePath = dir

	runTests := execTests

	// Filter for debugging
	//runTests = []execTestCase{}
	//for _, execTest := range execTests {
	//	if execTest.Input == "1 * 0" {
	//		runTests = append(runTests, execTest)
	//	}
	//}

	zs := NewZappacState("")
	for _, execTest := range runTests {
		func() {
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("%s (?): got\n\t%s\nexpected\n\t%s", execTest.Input, err, execTest.Expected)
				}
			}()

			start := time.Now()
			nodes, err := Parse(execTest.Input)
			elapsed := time.Since(start)

			if err != nil {
				if err.Error() == execTest.Expected {
					// Some errors are expected in the test cases
					t.Log(execTest.Input, fmt.Sprintf("OK in %s", elapsed))
					return
				}
				t.Errorf("%s (%s): got\n\t%+v", execTest.Input, elapsed, err)
				return
			}

			fmt.Printf("Execing %+v\n", nodes)
			result, err := zs.Exec(nodes, true)
			if err != nil {
				if err.Error() == execTest.Expected {
					// Some errors are expected in the test cases
					t.Log(execTest.Input, fmt.Sprintf("OK in %s", elapsed))
					return
				}

				t.Errorf("%s (%s): got\n\t%v\nexpected\n\t%v", execTest.Input, elapsed, err, execTest.Expected)
				return
			}

			if result != execTest.Expected {
				t.Errorf("%s (%s): got\n\t%s\nexpected\n\t%s", execTest.Input, elapsed, result, execTest.Expected)
				return
			}

			t.Log(execTest.Input, fmt.Sprintf("OK in %s", elapsed))
		}()
	}
}
