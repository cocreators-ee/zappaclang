package zappaclang

import (
	"fmt"
	"testing"
	"time"
)

type lexTest struct {
	name  string
	input string
	items []item
}

func mkItem(typ ItemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

var (
	tEOF    = mkItem(itemEOF, "")
	tLpar   = mkItem(itemLParen, "(")
	tRpar   = mkItem(itemRParen, ")")
	tSpace  = mkItem(itemSpace, " ")
	tEquals = mkItem(itemEquals, "=")
	tLShift = mkItem(itemLShift, "<<")
	tRShift = mkItem(itemRShift, ">>")
	tAdd    = mkItem(itemAdd, "+")
	tSub    = mkItem(itemSub, "-")
	tMult   = mkItem(itemMult, "*")
	tExp    = mkItem(itemExp, "**")
	tDiv    = mkItem(itemDiv, "/")
	tFdiv   = mkItem(itemFdiv, "//")
	tMod    = mkItem(itemMod, "%")
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"error", "<", []item{mkItem(itemError, "Unexpected <")}},

	{"space", " \t\r\n \t\t\r\n", []item{tSpace, tEOF}},
	{"variable", "$foo", []item{mkItem(itemVariable, "$foo"), tEOF}},
	{"variable with space around", "  \t$foo   \n", []item{tSpace, mkItem(itemVariable, "$foo"), tSpace, tEOF}},
	{"assign to variable", "$f_a_b_u_l_o_u_s=717", []item{mkItem(itemVariable, "$f_a_b_u_l_o_u_s"), tEquals, mkItem(itemNumber, "717"), tEOF}},
	{"assign with spaces", "$bar   =  b001", []item{mkItem(itemVariable, "$bar"), tSpace, tEquals, tSpace, mkItem(itemNumber, "b001"), tEOF}},
	{"lshift", "b001 << 10", []item{mkItem(itemNumber, "b001"), tSpace, tLShift, tSpace, mkItem(itemNumber, "10"), tEOF}},
	{"rshift", "0x7f>>1", []item{mkItem(itemNumber, "0x7f"), tRShift, mkItem(itemNumber, "1"), tEOF}},
	{"simple math", "3+1*2**3/4//2%3-1", []item{
		mkItem(itemNumber, "3"), tAdd, mkItem(itemNumber, "1"), tMult, mkItem(itemNumber, "2"), tExp, mkItem(itemNumber, "3"), tDiv,
		mkItem(itemNumber, "4"), tFdiv, mkItem(itemNumber, "2"), tMod, mkItem(itemNumber, "3"), tSub, mkItem(itemNumber, "1"), tEOF,
	}},

	{
		"calculation with variable", "$foo = 3 + $bar", []item{
			mkItem(itemVariable, "$foo"), tSpace, tEquals, tSpace, mkItem(itemNumber, "3"), tSpace, tAdd, tSpace, mkItem(itemVariable, "$bar"), tEOF,
		},
	},

	{"parenthesis", "((3+$bar)-1*3)//2", []item{
		tLpar,
		tLpar, mkItem(itemNumber, "3"), tAdd, mkItem(itemVariable, "$bar"), tRpar,
		tSub, mkItem(itemNumber, "1"), tMult, mkItem(itemNumber, "3"),
		tRpar,
		tFdiv, mkItem(itemNumber, "2"), tEOF,
	}},

	{"decimals", "12.3456", []item{mkItem(itemNumber, "12.3456"), tEOF}},

	{"dec", "dec(0755)", []item{mkItem(itemDec, "dec"), tLpar, mkItem(itemNumber, "0755"), tRpar, tEOF}},
	{"bin", "bin(1+2)", []item{mkItem(itemBin, "bin"), tLpar, mkItem(itemNumber, "1"), tAdd, mkItem(itemNumber, "2"), tRpar, tEOF}},
	{"hex", "hex( -7+b01 )", []item{mkItem(itemHex, "hex"), tLpar, tSpace, tSub, mkItem(itemNumber, "7"), tAdd, mkItem(itemNumber, "b01"), tSpace, tRpar, tEOF}},
	{"oct", "oct(0x77)", []item{mkItem(itemOct, "oct"), tLpar, mkItem(itemNumber, "0x77"), tRpar, tEOF}},

	{"load", "load(foo)", []item{mkItem(itemLoad, "load"), tLpar, mkItem(itemText, "foo"), tRpar, tEOF}},
	{"save", "save(bar_name)", []item{mkItem(itemSave, "save"), tLpar, mkItem(itemText, "bar_name"), tRpar, tEOF}},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	_, itemChan := lex(t.input)

	for {
		item, ok := <-itemChan
		if !ok {
			break
		}

		items = append(items, item)

		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}

	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		fmt.Printf("Lexing: %#v\n", test.input)

		start := time.Now()
		items := collect(&test)
		elapsed := time.Since(start)

		if !equal(items, test.items, false) {
			t.Errorf("%s (%s): got\n\t%+v\nexpected\n\t%v", test.name, elapsed, items, test.items)
			return
		}

		t.Log(test.name, fmt.Sprintf("OK in %s", elapsed))
	}
}
