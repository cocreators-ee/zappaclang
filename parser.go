package zappaclang

import (
	"errors"
	"fmt"
)

type parser struct {
	lexer       *lexer
	items       []item
	input       string
	parenthesis int
	pos         Pos
}

var (
	// ErrorUnexpectedEOF is given when there is an unexpected EOF
	ErrorUnexpectedEOF = errors.New("unexpected end of input")
	// ErrorInternal is used for unspecified internal errors
	ErrorInternal = errors.New("internal error while parsing")
)

func (p *parser) parse() (nodes []Node, err error) {
	lexer, items := lex(p.input)
	p.lexer = lexer
	p.parenthesis = 0

	nodes, err = p.readTokens(items)
	// TODO: Append item pos to err
	return
}

func (p *parser) nextItem(items chan item) (*item, error) {
	// If the next item has already been peeked at
	if p.pos < Pos(len(p.items)) {
		item := p.items[p.pos]
		p.pos++
		return &item, nil
	}

	for item := range items {
		// Skip spaces, they have no meaning for our parsing
		if item.typ == itemSpace {
			continue
		}

		if item.typ == itemEOF {
			if p.parenthesis > 0 {
				return nil, fmt.Errorf("unexpected end of input, there are unclosed parenthesis")
			}
			return nil, nil
		}

		if item.typ == itemError {
			return nil, fmt.Errorf("%s at pos %d", item.val, item.pos)
		}

		p.items = append(p.items, item)
		p.pos++

		return &item, nil
	}

	// Should never end up here, should get an itemEOF
	return nil, ErrorInternal
}

func (p *parser) peek(items chan item) (*item, error) {
	start := p.pos
	item, err := p.nextItem(items)

	// We want to back up a bit so this item will get scanned again next call to nextItem()
	p.pos = start

	return item, err
}

func isNodeType(node Node, types []NodeType) bool {
	t := node.Type()
	for _, _type := range types {
		if t == _type {
			return true
		}
	}

	return false
}

func isItemType(item *item, types []ItemType) bool {
	for _, _type := range types {
		if item.typ == _type {
			return true
		}
	}

	return false
}

func (p *parser) readTokens(items chan item) (nodes []Node, err error) {
	nodes = []Node{}
	for {
		// Read new item
		var itm *item
		itm, err = p.nextItem(items)
		if err != nil {
			return
		}

		if itm == nil {
			/*
				EOF
			*/

			// Parsing completed, check that last item makes sense
			if p.pos >= 1 {
				left := nodes[len(nodes)-1]
				validLeftTypes := append(valueNodes, NodeRParen)

				if !isNodeType(left, validLeftTypes) {
					err = ErrorUnexpectedEOF
				}
			}

			return
		} else if itm.typ == itemEquals {
			/*
				=
			*/
			if p.pos != 2 || nodes[0].Type() != NodeVariable {
				// Equals can only be used to assign to variables, so as the 2nd thing on the line
				err = fmt.Errorf("equals can only follow a variable name at the very start of the line. Ex: $foo = 1")
				return
			}

			// Replace original variable reference with an assignment
			target := nodes[0].(VariableNode)
			nodes[0] = newAssign(p.pos, target.Name)
		} else if itm.typ == itemVariable {
			/*
				$foo
			*/
			if p.pos != 1 {
				left := nodes[len(nodes)-1]

				validLeftTypes := append(operatorNodes, NodeLParen, NodeAssign)
				if len(nodes) == 1 {
					validLeftTypes = append(validLeftTypes, prefixNodes...)
				}

				if !isNodeType(left, validLeftTypes) {
					err = fmt.Errorf("unexpected %s at pos %d following %s", itm.val, itm.pos, left.String())
					return
				}
			}

			nodes = append(nodes, newVariable(p.pos, itm.val))

		} else if isItemType(itm, []ItemType{itemDec, itemBin, itemOct, itemHex}) {
			/*
				Set output mode: dec() bin() oct() hex()
			*/
			if p.pos != 1 {
				err = fmt.Errorf("unexpected %s at pos %d, setting output type must be the first thing you do", itm.val, itm.pos)
				return
			}

			nodes = append(nodes, newSetOutput(p.pos, itm.val))

		} else if isItemType(itm, operatorItems) {
			/*
				Operators: + - * ** / // & | ^ ~ % << >>
				(and negative numbers)
			*/
			// Special handling of - for negative numbers
			isNegativeNumber := false
			var peek *item = nil
			if itm.typ == itemSub {
				left := nodes[len(nodes)-1]
				// Check for 2 - 1 or $foo - 1
				if !isNodeType(left, valueNodes) {
					peek, err = p.peek(items)
					if err != nil {
						return
					}

					// Is the next item a number?
					if peek != nil && peek.typ == itemNumber {
						if p.pos == 1 {
							isNegativeNumber = true
						} else {
							// Check for various cases for negative number parsing
							/*
								2 + -1  // Operators
								abs(-1) // LParen
								2 + (-1 * 3)
								$foo = -1 // Prefixes
								dec(-7)
							*/
							validLeftTypes := append(operatorNodes, prefixNodes...)
							if isNodeType(left, validLeftTypes) {
								isNegativeNumber = true
							}
						}
					}
				}
			}

			if isNegativeNumber {
				// This is a negative number, do number validation
				value := fmt.Sprintf("-%s", peek.val)
				if p.pos != 1 {
					left := nodes[len(nodes)-1]
					validLeftTypes := append(operatorNodes, prefixNodes...)

					if !isNodeType(left, validLeftTypes) {
						err = fmt.Errorf("unexpected %s at pos %d, looks like a negative number that doesn't make sense here", value, itm.pos)
						return
					}
				}
				nodes = append(nodes, newNumber(p.pos, value, parseNumberSystem(peek.val)))
				p.pos++ // Skip peeked item, it's been parsed
			} else {
				// Is an operator valid here - typically needs a value on the left (and right, but that will be checked later), or rparen
				left := nodes[len(nodes)-1]
				validLeftTypes := append(valueNodes, NodeRParen)

				if !isNodeType(left, validLeftTypes) {
					err = fmt.Errorf("unexpected %s at pos %d, operators should follow numbers, variables, or closing parenthesis", itm.val, itm.pos)
					return
				}

				nodes = append(nodes, newOperator(p.pos, itm.val))
			}
		} else if itm.typ == itemNumber {
			/*
				Numbers:
				5  // Dec
				1.234
				0xff // Hex
				0775 // Oct
				b001 // Bin
			*/

			if p.pos != 1 {
				left := nodes[len(nodes)-1]
				validLeftTypes := append(operatorNodes, prefixNodes...)

				if !isNodeType(left, validLeftTypes) {
					err = fmt.Errorf("unexpected %s at pos %d, looks like a negative number that doesn't make sense here", itm.val, itm.pos)
					return
				}
			}

			// Number should look like a legitimate number from lexing, just need to figure out system
			nodes = append(nodes, newNumber(p.pos, itm.val, parseNumberSystem(itm.val)))
		} else if isItemType(itm, []ItemType{itemSave, itemLoad}) {
			/*
				save(name)
				load(name)
			*/

			invalidErr := fmt.Errorf("unexpected %s at pos %d, when used the input should be only: %s(name)", itm.val, itm.pos, itm.val)

			if p.pos != 1 {
				err = invalidErr
				return
			}

			// Validation is very fixed, but depends on the rest of the input having been read, so read until EOF
			for {
				var _itm *item
				_itm, err = p.nextItem(items)
				if err != nil {
					return
				}
				if _itm == nil {
					break
				}
			}

			if len(p.items) != 4 {
				err = invalidErr
				return
			}

			// save|load followed by ( name )
			if p.items[1].typ != itemLParen || p.items[2].typ != itemText || p.items[3].typ != itemRParen {
				err = invalidErr
				return
			}

			nodes = append(nodes, newDiskOperation(p.pos, itm.val, p.items[2].val))

			// Since we just consumed all the items, we need to bail out here or get an internal error
			return
		} else if itm.typ == itemLParen {
			/*
				(
				abs(
				dec(
			*/
			// Allowed following abs() dec() hex() bin() oct() = ( and operators
			if p.pos != 1 {
				left := nodes[len(nodes)-1]
				validLeftTypes := append(operatorNodes, prefixNodes...)
				validLeftTypes = append(validLeftTypes, NodeAbs)
				if !isNodeType(left, validLeftTypes) {
					err = fmt.Errorf("unexpected ( at pos %d, should be following abs, dec, hex, bin, oct, =, operators, or other (s", itm.pos)
					return
				}
			}

			// Increase parenthesis level
			p.parenthesis++

			nodes = append(nodes, newLParen(p.pos))
		} else if itm.typ == itemRParen {
			/*
				abs(-2)
				(1 + 2)
				((2 + 3) * 5)
			*/

			// Closing parenthesis when none are open
			if p.parenthesis == 0 { // Also explicitly checks for pos > 1
				err = fmt.Errorf("unexpected ) at pos %d, no parenthesis open", itm.pos)
				return
			}

			left := nodes[len(nodes)-1]
			validLeftTypes := append(valueNodes, NodeRParen)

			if !isNodeType(left, validLeftTypes) {
				err = fmt.Errorf("unexpected ) at pos %d, should be following numbers, variables, or other )s", itm.pos)
				return
			}

			// Decrease parenthesis level
			p.parenthesis--

			nodes = append(nodes, newRParen(p.pos))
		} else if itm.typ == itemAbs {
			/*
				abs()
			*/
			if p.pos != 1 {
				left := nodes[len(nodes)-1]
				validLeftTypes := append(operatorNodes, prefixNodes...)

				if !isNodeType(left, validLeftTypes) {
					err = fmt.Errorf("unexpected abs() at pos %d, may follow operators, (, or =", itm.pos)
					return
				}
			}

			nodes = append(nodes, newAbs(p.pos))
		}
	}
}

// Parse any zappac lang string
func Parse(input string) (nodes []Node, err error) {
	p := &parser{
		input: input,
	}

	nodes, err = p.parse()
	return
}
