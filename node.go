package zappaclang

import (
	"fmt"
	"strconv"
	"strings"
)

// NodeType identifies the different types of nodes
type NodeType int

// Node is an individual node in the string
type Node interface {
	Type() NodeType
	String() string
	Position() Pos
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

// Position returns itself similarly for easy embedding in a Node
func (p Pos) Position() Pos {
	return p
}

const (
	// NodeEOF is the end of input
	NodeEOF NodeType = iota
	// NodeAssign is for $foo =
	NodeAssign
	// NodeLParen is for (
	NodeLParen
	// NodeRParen is for )
	NodeRParen
	// NodeNumber is for 123, 0.123, 0xff, b001, and 0755
	NodeNumber
	// NodeVariable is for $foo
	NodeVariable
	// NodeAdd is for +
	NodeAdd
	// NodeSub is for -
	NodeSub
	// NodeMult is for *
	NodeMult
	// NodeExp is for **
	NodeExp
	// NodeDiv is for /
	NodeDiv
	// NodeFdiv is for //
	NodeFdiv
	// NodeAnd is for &
	NodeAnd
	// NodeOr is for |
	NodeOr
	// NodeXor is for ^
	NodeXor
	// NodeInv is for ~
	NodeInv
	// NodeMod is for %
	NodeMod
	// NodeLShift is for <<
	NodeLShift
	// NodeRShift is for >>
	NodeRShift
	// NodeAbs is for abs()
	NodeAbs
	// NodeSetOutput is for dec() bin() oct() hex()
	NodeSetOutput
	// NodeSave is for save(foo)
	NodeSave
	// NodeLoad is for load(foo)
	NodeLoad
	// NodeClear is for clear()
	NodeClear
)

//go:generate stringer -type=NodeType

// OperatorNodes are operators between values
var OperatorNodes = []NodeType{
	NodeAdd,
	NodeSub,
	NodeMult,
	NodeExp,
	NodeDiv,
	NodeFdiv,
	NodeAnd,
	NodeOr,
	NodeXor,
	NodeInv,
	NodeMod,
	NodeLShift,
	NodeRShift,
}

var operatorMap = map[string]NodeType{
	"+":  NodeAdd,
	"-":  NodeSub,
	"*":  NodeMult,
	"**": NodeExp,
	"/":  NodeDiv,
	"//": NodeFdiv,
	"&":  NodeAnd,
	"|":  NodeOr,
	"^":  NodeXor,
	"~":  NodeInv,
	"%":  NodeMod,
	"<<": NodeLShift,
	">>": NodeRShift,
}

var diskOperationMap = map[string]NodeType{
	"save": NodeSave,
	"load": NodeLoad,
}

var numberSystemMap = map[string]NumberSystem{
	"dec": Dec,
	"bin": Bin,
	"hex": Hex,
	"oct": Oct,
}

// FunctionNodes are any functions
var FunctionNodes = []NodeType{
	NodeAbs,
	NodeSetOutput,
	NodeSave,
	NodeLoad,
	NodeClear,
}

// Nodes that can be prefixes to most values
var prefixNodes = []NodeType{
	NodeLParen,
	NodeSetOutput,
	NodeAssign,
}

// ValueNodes are values that can be evaluated as values
var ValueNodes = []NodeType{
	NodeNumber,
	NodeVariable,
}

// NumberSystem defines which of the number systems should be used
type NumberSystem int

const (
	// Dec imal
	Dec NumberSystem = iota
	// Hex adecimal
	Hex
	// Bin ary
	Bin
	// Oct al
	Oct
)

//go:generate stringer -type=NumberSystem

func parseNumberSystem(number string) NumberSystem {
	if number[0] == 'b' || number[0] == 'B' {
		return Bin
	}
	if number[0] == '0' {
		if len(number) == 1 {
			// Literal "0"
			return Dec
		}
		if number[1] == 'x' || number[1] == 'X' {
			return Hex
		}
		if number[1] == '.' {
			return Dec
		}
		return Oct
	}
	return Dec
}

// AssignNode $foo =
type AssignNode struct {
	NodeType
	Pos
	Target string
}

func (an AssignNode) String() string {
	return fmt.Sprintf("%s =", an.Target)
}

func newAssign(pos Pos, target string) AssignNode {
	return AssignNode{
		NodeType: NodeAssign,
		Pos:      pos,
		Target:   target,
	}
}

// VariableNode $foo
type VariableNode struct {
	NodeType
	Pos
	Name string
}

func (vn VariableNode) String() string {
	return vn.Name
}

func newVariable(pos Pos, name string) VariableNode {
	return VariableNode{
		NodeType: NodeVariable,
		Pos:      pos,
		Name:     name,
	}
}

// SetOutputNode dec() oct() hex() bin()
type SetOutputNode struct {
	NodeType
	Pos
	Output NumberSystem
}

func (vn SetOutputNode) String() string {
	return vn.Output.String()
}

func newSetOutput(pos Pos, output string) SetOutputNode {
	return SetOutputNode{
		NodeType: NodeSetOutput,
		Pos:      pos,
		Output:   numberSystemMap[output],
	}
}

// NumberNode 123, 0.123, 0xff, b001, and 0755
type NumberNode struct {
	NodeType
	Pos
	Value  string
	System NumberSystem
}

func (nn NumberNode) String() string {
	return nn.Value
}

func (nn NumberNode) toFloat64() (float64, error) {
	if nn.System == Dec {
		f64, err := strconv.ParseFloat(nn.Value, 64)
		return f64, err
	} else if nn.System == Oct || nn.System == Hex {
		i64, err := strconv.ParseInt(nn.Value, 0, 64)
		return float64(i64), err
	} else { // Bin
		i64, err := strconv.ParseInt("0"+nn.Value, 0, 64)
		return float64(i64), err
	}
}

func newNumber(pos Pos, value string, system NumberSystem) NumberNode {
	return NumberNode{
		NodeType: NodeNumber,
		Pos:      pos,
		Value:    strings.ToLower(value),
		System:   system,
	}
}

// OperatorNode + - * ** / // & | ^ ~ % << >>
type OperatorNode struct {
	NodeType
	Pos
	Operator string
}

func (on OperatorNode) String() string {
	return on.Operator
}

func newOperator(pos Pos, op string) OperatorNode {
	return OperatorNode{
		NodeType: operatorMap[op],
		Pos:      pos,
		Operator: op,
	}
}

// DiskOperationNode save(name) load(name)
type DiskOperationNode struct {
	NodeType
	Pos
	Operation string
	Profile   string
}

func (don DiskOperationNode) String() string {
	return fmt.Sprintf("%s(%s)", don.Operation, don.Profile)
}

func newDiskOperation(pos Pos, op string, profile string) DiskOperationNode {
	return DiskOperationNode{
		NodeType:  diskOperationMap[op],
		Pos:       pos,
		Operation: op,
		Profile:   profile,
	}
}

// LParenNode (
type LParenNode struct {
	NodeType
	Pos
}

func (lpn LParenNode) String() string {
	return "("
}

func newLParen(pos Pos) LParenNode {
	return LParenNode{
		NodeType: NodeLParen,
		Pos:      pos,
	}
}

// RParenNode )
type RParenNode struct {
	NodeType
	Pos
}

func (lpn RParenNode) String() string {
	return ")"
}

func newRParen(pos Pos) RParenNode {
	return RParenNode{
		NodeType: NodeRParen,
		Pos:      pos,
	}
}

// AbsNode abs()
type AbsNode struct {
	NodeType
	Pos
}

func (an AbsNode) String() string {
	return "abs"
}

func newAbs(pos Pos) AbsNode {
	return AbsNode{
		NodeType: NodeAbs,
		Pos:      pos,
	}
}

// EOFNode EOF
type EOFNode struct {
	NodeType
	Pos
}

func (e EOFNode) String() string {
	return ""
}

func newEOF(pos Pos) EOFNode {
	return EOFNode{
		NodeType: NodeEOF,
		Pos:      pos,
	}
}

// ClearNode clear()
type ClearNode struct {
	NodeType
	Pos
}

func (c ClearNode) String() string {
	return "clear()"
}

func newClear(pos Pos) ClearNode {
	return ClearNode{
		NodeType: NodeClear,
		Pos:      pos,
	}
}
