// Inspired by go text/template/parse/lex and Rob Pike's "Lexical Scanning in Go"
// https://cs.opensource.google/go/go/+/master:src/text/template/parse/lex.go

package zappaclang

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const debugLex = false

/*
$foo = 1
$foo = $foo ** 2
$foo * 2
(-1 + 2) -3 / 4 // 5 + 0xff-0775-b001
6 & 9
9 | 0
0xff << 10
0xaa >> 1
~9
2 ^ 3 % 7
abs(-5) - 5
save(foo)
load(bar)
dec(b111)
hex(0400)
bin(123 ** 2)
oct(5+$foo)

---

$foo = variable
11, -195, 0xff, 0777, b100 = number
( ) = parenthesis
+ = add
- = sub
* = mult
** = exp
/ = div
// = fdiv
& = bitwise and
| = bitwise or
^ = bitwise xor
~ = bitwise inversion
% = modulus
<< = lshift
>> = rshift
abs = absolute
= = equals
+= plus equals
-= minus equals
*= mult equals
/= div equals
//= fdiv equals
**= exp equals
&= and equals
|= or equals
^= xor equals
~= invert equals
%= mod equals
<<= lshift equals
>>= rshift equals
save = save
load = load
dec = decimal
hex = hexadecimal
bin = binary
oct = octal
*/

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

// ItemType identifies the type of lex items.
type ItemType int

const (
	itemError    ItemType = iota // error occurred; value is text of error
	itemEOF                      // End of input
	itemEquals                   // '=', assignment
	itemSpace                    // whitespace
	itemLParen                   // '('
	itemRParen                   // ')'
	itemNumber                   // numbers like 135, 1.23, 0x7f, b0100, 0755
	itemVariable                 // variable starting with '$', e.g. '$hello'
	itemAdd                      // + add
	itemSub                      // - substract
	itemMult                     // * multiply
	itemExp                      // ** exponent
	itemDiv                      // / division
	itemFdiv                     // // floor-div
	itemAnd                      // &	bitwise and
	itemOr                       // | bitwise or
	itemXor                      // ^ bitwise xor
	itemInv                      // ~ bitwise inversion
	itemMod                      // % modulus
	itemLShift                   // << left shift
	itemRShift                   // >> right shift
	// TODO: += .. >>=
	// The plain text things rely on being after itemText for simplified stringification
	itemText // plain text
	itemAbs  // abs() - calculate absolute value
	// The following can only exist at the start of the line
	itemSave // save state
	itemLoad // load state
	itemDec  // dec()
	itemHex  // hex()
	itemBin  // bin()
	itemOct  // oct()
)

var operatorItems = []ItemType{
	itemAdd,
	itemSub,
	itemMult,
	itemExp,
	itemDiv,
	itemFdiv,
	itemAnd,
	itemOr,
	itemXor,
	itemInv,
	itemMod,
	itemLShift,
	itemRShift,
}

//go:generate stringer -type=ItemType

const eof = -1

const (
	whitespaceChars = " \t\r\n"
	letters         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits          = "0123456789"
	hexadecimal     = digits + "abcdefABCDEF"
	binary          = "01"
	alnum           = letters + digits
)

type item struct {
	typ ItemType // The type of this item.
	pos Pos      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case i.typ == itemLParen || i.typ == itemRParen:
		return i.val
	case i.typ >= itemAdd && i.typ < itemText:
		return i.val
	case i.typ > itemText:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("<%s>%.10q...", i.typ, i.val)
	}
	name := strings.TrimPrefix(i.typ.String(), "item")
	return fmt.Sprintf("<%s>%q", name, i.val)
}

// lexer holds the state of the scanner.
type lexer struct {
	input string // the string being scanned
	pos   Pos    // current position in the input
	start Pos    // start position of this item
	len   Pos
	atEOF bool      // we have hit the end of input and returned eof
	items chan item // channel to send items through
}

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.atEOF = true
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += Pos(w)
	return r
}

// peek returns but does not consume the next rune in the input.
/*
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}
*/

// backup steps back one rune.
func (l *lexer) backup() {
	if !l.atEOF && l.pos > 0 {
		_, w := utf8.DecodeLastRuneInString(l.input[:l.pos])
		l.pos -= Pos(w)
	}
}

// thisItem returns the item at the current input point with the specified type
// and advances the input.
func (l *lexer) thisItem(t ItemType) item {
	i := item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
	return i
}

// emit passes the trailing text as an item back to the parser.
func (l *lexer) emit(t ItemType) stateFn {
	return l.emitItem(l.thisItem(t))
}

// emitItem passes the specified item to the parser.
func (l *lexer) emitItem(i item) stateFn {
	if debugLex {
		fmt.Printf("LEX -> %s\n", i.val)
	}
	l.items <- i
	return nil
}

// ignore skips over the pending input before this point.
// It tracks newlines in the ignored text, so use it only
// for text that is skipped without calling l.next.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...any) stateFn {
	item := item{itemError, l.start, fmt.Sprintf(format, args...)}
	l.emitItem(item)
	l.start = 0
	l.pos = 0
	l.input = l.input[:0]
	return nil
}

func (l *lexer) run() {
	for state := lexBase; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) debug(location string) {
	if debugLex {
		fmt.Printf("LEX %s @ %d\n", location, l.pos)
	}
}

// lex creates a new scanner for the input string.
func lex(input string) (*lexer, chan item) {
	l := &lexer{
		input: input,
		len:   Pos(len(input)),
		items: make(chan item),
	}
	go l.run()
	return l, l.items
}

type lexMapItem struct {
	key     string
	stateFn stateFn
}

func lexBase(l *lexer) stateFn {
	l.debug("base")

	// TODO: Would be nice if this could be on higher level and wouldn't need to be redefined
	// ... but since these functions have references to lexBase, which uses this map, it apparently doesn't work
	lexMap := []lexMapItem{
		{"$", lexVariable},
		{"(", lexLParen},
		{")", lexRParen},
		{"=", lexEquals},
		{"+", lexAdd},
		{"-", lexSub},
		{"**", lexExp},
		{"*", lexMult},
		{"//", lexFdiv},
		{"/", lexDiv},
		{"&", lexAnd},
		{"|", lexOr},
		{"^", lexXor},
		{"~", lexInv},
		{"%", lexMod},
		{"<<", lexLShift},
		{">>", lexRShift},
	}

	// Any leading whitespace is condensed to one
	l.acceptRun(whitespaceChars)
	if l.pos > l.start {
		l.ignore()
		l.emitItem(item{itemSpace, l.start, " "})
	}

	for _, lexMapItem := range lexMap {
		if strings.HasPrefix(l.input[l.pos:], lexMapItem.key) {
			return lexMapItem.stateFn
		}
	}

	if l.accept("b") && l.accept(binary) {
		l.backup()
		l.backup()
		return lexNumber
	}

	if l.accept(digits) {
		l.backup()
		return lexNumber
	}

	if l.accept(letters) {
		l.backup()
		return lexText
	}

	if l.pos == l.len {
		l.emit(itemEOF)
		return nil
	}

	// Any other options?
	return l.errorf("Unexpected %c", l.next())
}

func lexVariable(l *lexer) stateFn {
	l.debug("variable")

	// Must start with a $
	l.accept("$")

	validChars := letters + "_"
	l.acceptRun(validChars)
	if l.pos > l.start {
		l.emit(itemVariable)
	}

	return lexBase
}

func lexText(l *lexer) stateFn {
	l.debug("text")

	// Must start with a letter, then can be followed by letters and underscores
	l.accept(letters)
	l.acceptRun(letters + "_")

	item := l.thisItem(itemText)

	if item.val == "save" {
		item.typ = itemSave
		l.emitItem(item)
	} else if item.val == "load" {
		item.typ = itemLoad
		l.emitItem(item)
	} else if item.val == "abs" {
		item.typ = itemAbs
		l.emitItem(item)
	} else if item.val == "dec" {
		item.typ = itemDec
		l.emitItem(item)
	} else if item.val == "hex" {
		item.typ = itemHex
		l.emitItem(item)
	} else if item.val == "bin" {
		item.typ = itemBin
		l.emitItem(item)
	} else if item.val == "oct" {
		item.typ = itemOct
		l.emitItem(item)
	} else {
		l.emitItem(item)
	}

	return lexBase
}

func lexNumber(l *lexer) stateFn {
	l.debug("number")

	if l.accept("b") {
		l.acceptRun(binary)
		l.emit(itemNumber)
	} else if l.accept("0") && l.accept("xX") {
		l.acceptRun(hexadecimal)
		l.emit(itemNumber)
	} else if l.accept(digits) {
		decimal := false

		for {
			if !decimal && l.accept(".") {
				decimal = true
			} else if !l.accept(digits) {
				break
			}
		}

		l.emit(itemNumber)
	}

	return lexBase
}

func lexLParen(l *lexer) stateFn {
	l.debug("lparen")

	l.accept("(")
	l.emit(itemLParen)
	return lexBase
}

func lexRParen(l *lexer) stateFn {
	l.debug("rparen")

	l.accept(")")
	l.emit(itemRParen)
	return lexBase
}

func lexEquals(l *lexer) stateFn {
	l.debug("equals")

	l.accept("=")
	l.emit(itemEquals)
	return lexBase
}

func lexAdd(l *lexer) stateFn {
	l.debug("add")

	l.accept("+")
	l.emit(itemAdd)
	return lexBase
}

func lexSub(l *lexer) stateFn {
	l.debug("sub")

	l.accept("-")

	l.emit(itemSub)
	return lexBase
}

func lexExp(l *lexer) stateFn {
	l.debug("exp")

	l.accept("*")
	l.accept("*")

	l.emit(itemExp)
	return lexBase
}

func lexMult(l *lexer) stateFn {
	l.debug("mult")

	l.accept("*")

	l.emit(itemMult)
	return lexBase
}

func lexFdiv(l *lexer) stateFn {
	l.debug("fdiv")

	l.accept("/")
	l.accept("/")

	l.emit(itemFdiv)
	return lexBase
}

func lexDiv(l *lexer) stateFn {
	l.debug("div")

	l.accept("/")

	l.emit(itemDiv)
	return lexBase
}

func lexAnd(l *lexer) stateFn {
	l.debug("and")

	l.accept("&")

	l.emit(itemAnd)
	return lexBase
}

func lexOr(l *lexer) stateFn {
	l.debug("or")

	l.accept("|")

	l.emit(itemOr)
	return lexBase
}

func lexXor(l *lexer) stateFn {
	l.debug("xor")

	l.accept("^")

	l.emit(itemXor)
	return lexBase
}

func lexInv(l *lexer) stateFn {
	l.debug("inv")

	l.accept("~")

	l.emit(itemInv)
	return lexBase
}

func lexMod(l *lexer) stateFn {
	l.debug("mod")

	l.accept("%")

	l.emit(itemMod)
	return lexBase
}

func lexLShift(l *lexer) stateFn {
	l.debug("lshift")

	l.accept("<")
	l.accept("<")

	l.emit(itemLShift)
	return lexBase
}

func lexRShift(l *lexer) stateFn {
	l.debug("rshift")

	l.accept(">")
	l.accept(">")

	l.emit(itemRShift)
	return lexBase
}
