package zappaclang

import (
	"fmt"
	"testing"
	"time"
)

type simpleNode struct {
	typ NodeType
	val string
}

type parserTest struct {
	name  string
	input string
	nodes []simpleNode
}

var parserTests = []parserTest{
	{"empty", "", []simpleNode{}},
	{"abc", "abc", []simpleNode{}},
	{"save", "save(foobar)", []simpleNode{{typ: NodeSave, val: "save(foobar)"}}},
	{"load", "load(foobar)", []simpleNode{{typ: NodeLoad, val: "load(foobar)"}}},
	{"bin", "bin(16 ** 2)", []simpleNode{
		{typ: NodeSetOutput, val: "Bin"},
		{typ: NodeLParen, val: "("},
		{typ: NodeNumber, val: "16"},
		{typ: NodeExp, val: "**"},
		{typ: NodeNumber, val: "2"},
		{typ: NodeRParen, val: ")"},
	}},
	{"complex", "$foo = ((1 - 2) ** abs(-7)) // b100", []simpleNode{
		{typ: NodeAssign, val: "$foo ="},
		{typ: NodeLParen, val: "("},
		{typ: NodeLParen, val: "("},
		{typ: NodeNumber, val: "1"},
		{typ: NodeSub, val: "-"},
		{typ: NodeNumber, val: "2"},
		{typ: NodeRParen, val: ")"},
		{typ: NodeExp, val: "**"},
		{typ: NodeAbs, val: "abs"},
		{typ: NodeLParen, val: "("},
		{typ: NodeNumber, val: "-7"},
		{typ: NodeRParen, val: ")"},
		{typ: NodeRParen, val: ")"},
		{typ: NodeFdiv, val: "//"},
		{typ: NodeNumber, val: "b100"},
	}},
	{"no spaces", "(-1+2)-3/abs(4//5)+0xff-0775-b001", []simpleNode{
		{typ: NodeLParen, val: "("},
		{typ: NodeNumber, val: "-1"},
		{typ: NodeAdd, val: "+"},
		{typ: NodeNumber, val: "2"},
		{typ: NodeRParen, val: ")"},
		{typ: NodeSub, val: "-"},
		{typ: NodeNumber, val: "3"},
		{typ: NodeDiv, val: "/"},
		{typ: NodeAbs, val: "abs"},
		{typ: NodeLParen, val: "("},
		{typ: NodeNumber, val: "4"},
		{typ: NodeFdiv, val: "//"},
		{typ: NodeNumber, val: "5"},
		{typ: NodeRParen, val: ")"},
		{typ: NodeAdd, val: "+"},
		{typ: NodeNumber, val: "0xff"},
		{typ: NodeSub, val: "-"},
		{typ: NodeNumber, val: "0775"},
		{typ: NodeSub, val: "-"},
		{typ: NodeNumber, val: "b001"},
	}},
}

func parsedEqual(i1 []Node, i2 []simpleNode, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}

	for k := range i1 {
		if i1[k].Type() != i2[k].typ {
			return false
		}
		if i1[k].String() != i2[k].val {
			return false
		}
	}
	return true
}

func TestParse(t *testing.T) {
	for _, test := range parserTests {
		fmt.Printf("Parsing: %#v\n", test.input)

		start := time.Now()
		nodes, err := Parse(test.input)
		elapsed := time.Since(start)

		// eofPos := Pos(0)
		if nodes[len(nodes)-1].Type() == NodeEOF {
			// eofPos = nodes[len(nodes)-1].Position()
			nodes = nodes[:len(nodes)-1]
		}

		expected := []string{}
		for _, si := range test.nodes {
			expected = append(expected, si.val)
		}

		if err != nil {
			t.Errorf("%s (%s): got\n\t%v\nexpected\n\t%v", test.name, elapsed, err, expected)
			return
		}

		if !parsedEqual(nodes, test.nodes, false) {
			t.Errorf("%s (%s): got\n\t%+v\nexpected\n\t%v", test.name, elapsed, nodes, expected)
			return
		}

		// fmt.Printf("len() %d EOF @ %d\n", len(test.input), eofPos)

		t.Log(test.name, fmt.Sprintf("OK in %s", elapsed))
	}
}
