package zappaclang

import (
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"strconv"

	"gopkg.in/yaml.v3"
)

// StoragePath gets updated to the base path of Zappac state
var StoragePath = "."

var emptyNumber = newNumber(-1, "", Dec)

// OnSaveCallback is the type of the OnSave callback
type OnSaveCallback func()

// ZappacState contains the state for Zappac
type ZappacState struct {
	Variables map[string]NumberNode `yaml:"variables"`
	OnSave    OnSaveCallback        `yaml:"-"`
}

func getProfileFile(profile string) string {
	return fmt.Sprintf("%s/%s.json", StoragePath, profile)
}

func (zs *ZappacState) load(profile string) string {
	fp := getProfileFile(profile)

	contents, err := os.ReadFile(fp)
	if err != nil {
		return err.Error()
	}

	err = yaml.Unmarshal(contents, &zs)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("Loaded %s", profile)
}

func (zs *ZappacState) clear() {
	zs.Variables = map[string]NumberNode{}
}

func (zs *ZappacState) save(profile string) string {
	fp := getProfileFile(profile)

	contents, err := yaml.Marshal(&zs)
	if err != nil {
		return err.Error()
	}

	err = os.MkdirAll(StoragePath, 0o700)
	if err != nil {
		return err.Error()
	}

	err = os.WriteFile(fp, contents, 0o600)
	if err != nil {
		return err.Error()
	}

	zs.OnSave()

	return fmt.Sprintf("Saved %s", profile)
}

func findNext(nodes []Node, types []NodeType) int {
	for idx := 0; idx < len(nodes); idx++ {
		for _, typ := range types {
			if nodes[idx].Type() == typ {
				return idx
			}
		}
	}

	return -1
}

// Find closing parenthesis for the LParen
func findClosing(nodes []Node) int {
	parenthesis := 0

	for idx := 0; idx < len(nodes); idx++ {
		n := nodes[idx]
		typ := n.Type()
		if typ == NodeLParen {
			parenthesis++
		} else if typ == NodeRParen {
			parenthesis--
		}

		if parenthesis == 0 {
			return idx
		}
	}

	return -1
}

func replace(nodes []Node, start, end int, replace Node) []Node {
	// orig := fmt.Sprintf("%+v", nodes)
	newNodes := []Node{}
	newNodes = append(newNodes, nodes[0:start]...)
	newNodes = append(newNodes, replace)
	newNodes = append(newNodes, nodes[end+1:]...)
	// upda := fmt.Sprintf("%+v", newNodes)
	// fmt.Printf("%s -> %s\n", orig, upda)
	return newNodes
}

func (zs *ZappacState) readValue(node Node) (NumberNode, error) {
	typ := node.Type()
	if typ == NodeVariable {
		nv, _ := node.(VariableNode)
		val, ok := zs.Variables[nv.Name]
		if !ok {
			return emptyNumber, fmt.Errorf("unknown variable %s", nv.Name)
		}
		return val, nil
	}

	// typ == NodeNumber
	n, _ := node.(NumberNode)
	return n, nil
}

func (zs *ZappacState) calculate(left Node, op OperatorNode, right Node) (NumberNode, error) {
	// Each left or right can be a variable reference, or a number
	leftNum, err := zs.readValue(left)
	if err != nil {
		return emptyNumber, err
	}

	l, err := leftNum.toFloat64()
	if err != nil {
		return emptyNumber, err
	}

	rightNum, err := zs.readValue(right)
	if err != nil {
		return emptyNumber, err
	}

	r, err := rightNum.toFloat64()
	if err != nil {
		return emptyNumber, err
	}

	opType := op.Type()
	var result float64
	if opType == NodeAdd {
		result = l + r
	} else if opType == NodeSub {
		result = l - r
	} else if opType == NodeMult {
		result = l * r
	} else if opType == NodeExp {
		result = math.Pow(l, r)
	} else if opType == NodeDiv {
		result = l / r
	} else if opType == NodeFdiv {
		result = math.Floor(l / r)
	} else if opType == NodeAnd {
		result = l - r
	} else if opType == NodeOr {
		result = l - r
	} else if opType == NodeXor {
		result = l - r
	} else if opType == NodeInv {
		result = l - r
	} else if opType == NodeMod {
		result = math.Mod(l, r)
	} else if opType == NodeLShift {
		result = float64(int64(l) << int64(r))
	} else if opType == NodeRShift {
		result = float64(int64(l) >> int64(r))
	} else {
		return emptyNumber, fmt.Errorf("unknown operation %s", op)
	}

	resultStr := strconv.FormatFloat(result, 'f', -1, 64)
	// fmt.Printf("%s %s %s = %s\n", left, op, right, resultStr)

	return newNumber(-1, resultStr, Dec), nil
}

func (zs *ZappacState) calculateNodes(nodes []Node, operatorPos int) (result []Node, err error) {
	op, _ := nodes[operatorPos].(OperatorNode)

	left := operatorPos - 1
	right := operatorPos + 1

	// Calculate it away and replace in node tree
	value, err := zs.calculate(nodes[left], op, nodes[right])
	if err != nil {
		return
	}
	result = replace(nodes, left, right, value)
	return
}

// (parenthesis & exponent) (multiply & divide) (add & substract)
func (zs *ZappacState) pemdas(nodes []Node) (output string, err error) {
	// fmt.Printf("pemdas %+v\n", nodes)

	output = ""

	var next int
	for {
		if err != nil {
			return
		}

		// If the node list is only 1 item, it must be a number
		if len(nodes) == 1 {
			output = nodes[0].String()
			return
		}

		// Parenthesis & exponent
		next = findNext(nodes, []NodeType{NodeLParen, NodeExp})
		if next != -1 {
			node := nodes[next]
			typ := node.Type()

			if typ == NodeLParen {
				// Need to find closing parenthesis
				closing := next + findClosing(nodes[next:])

				// Calculate it away and replace in node tree
				var result string
				// fmt.Printf("Going in %d-%d of %+v\n", next, closing, nodes)
				result, err = zs.pemdas(nodes[next+1 : closing])
				nodes = replace(nodes, next, closing, newNumber(-1, result, Dec))
			} else if typ == NodeExp {
				nodes, err = zs.calculateNodes(nodes, next)
			}
			continue
		}

		// multiply & divide (and // % & | ^ ~ << >>)
		next = findNext(nodes, []NodeType{NodeMult, NodeDiv, NodeFdiv, NodeMod, NodeAnd, NodeOr, NodeXor, NodeInv, NodeLShift, NodeRShift})
		if next != -1 {
			nodes, err = zs.calculateNodes(nodes, next)
			continue
		}

		next = findNext(nodes, []NodeType{NodeAdd, NodeSub})
		if next != -1 {
			nodes, err = zs.calculateNodes(nodes, next)
			continue
		}

		// Cut out EOF cleanly
		if nodes[1].Type() == NodeEOF {
			nodes = []Node{nodes[0]}
			continue
		}

		fmt.Printf("Unexpected end of pemdas: %+v\n", nodes)
		return "ERROR", err
	}
}

// Exec executes logic from parsed nodes
func (zs *ZappacState) Exec(nodes []Node, updateVariables bool) (string, error) {
	// Is there anything to do?
	if len(nodes) == 0 {
		// TODO: Execute previous calculation again
		return "", nil
	}

	firstType := nodes[0].Type()
	targetVariable := ""
	outputSystem := Dec // TODO: Autodetect

	if firstType == NodeSetOutput {
		setOutput, _ := nodes[0].(SetOutputNode)
		outputSystem = setOutput.Output
		nodes = nodes[1:]
	} else if firstType == NodeAssign {
		assign, _ := nodes[0].(AssignNode)
		if updateVariables {
			targetVariable = assign.Target
		}
		nodes = nodes[1:]
	} else if firstType == NodeClear {
		zs.clear()
		return "Cleared state", nil
	} else if firstType == NodeSave {
		operation, _ := nodes[0].(DiskOperationNode)
		msg := zs.save(operation.Profile)
		return msg, nil
	} else if firstType == NodeLoad {
		operation, _ := nodes[0].(DiskOperationNode)
		msg := zs.load(operation.Profile)
		return msg, nil
	}

	result, err := zs.pemdas(nodes)
	if err == nil {
		if outputSystem != Dec {
			// TODO: Convert
			_ = outputSystem
		}

		if targetVariable != "" {
			zs.Variables[targetVariable] = newNumber(-1, result, parseNumberSystem(result))
		}
	}

	return result, err
}

// NewZappacState initializes a new ZappacState instance and loads existing state
func NewZappacState(name string) *ZappacState {
	zs := &ZappacState{
		Variables: map[string]NumberNode{},
		OnSave:    func() {},
	}

	zs.load(name)

	return zs
}

func init() {
	base := "."
	if runtime.GOOS == "windows" {
		base = path.Join(os.Getenv("APPDATA"), "zappac")
	} else {
		base = path.Join(os.Getenv("HOME"), ".config", "zappac")
	}

	StoragePath = base
}
