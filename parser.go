package main

import (
	"fmt"
	"regexp"
	"errors"
	"strings"
	"strconv"
)


// ----------------------------------------------------------------------------
// TOKENS
// A string is parsed into a list of tokens.
// ----------------------------------------------------------------------------
type TokenName string
const (
	IDENTIFIER TokenName = "IDENTIFIER"
	STRING_LITERAL = "STRING_LITERAL"
	INTEGER_LITERAL = "INTEGER_LITERAL"
	FLOAT_LITERAL = "FLOAT_LITERAL"
	WHITESPACE = "WHITESPACE"

	ITEM_SEPERATOR = "ITEM_SEPERATOR"
	CALL_IN = "CALL_IN"
	CALL_OUT = "CALL_OUT"
	ARG_LIST_IN = "ARG_LIST_IN"
	ARG_LIST_OUT = "ARG_LIST_OUT"
	BLOCK_IN = "BLOCK_IN"
	BLOCK_OUT = "BLOCK_OUT"

	// The root AST node
	ROOT = "ROOT"
	CALL_EXPRESSION = "CALL_EXPRESSION"
	ARG_LIST_EXPRESSION = "ARG_LIST_EXPRESSION"
	BLOCK_EXPRESSION = "BLOCK_EXPRESSION"
)

type Token struct {
	Name TokenName
	Shape *regexp.Regexp
	Data []string

	// Called before a new ast frame is created
	HookPreNew func([]string, *AstFrame, string, int) (map[string]interface{}, *AstFrame, error)
	// Called after a new ast frame is created
	HookPostNew func([]string, *AstFrame, string, int) (*AstFrame, error)
}
var EMPTY_DATA map[string]interface{} = nil

var TOKENS = []Token{


	Token{
		Name: WHITESPACE,
		Shape: regexp.MustCompile(`^[\n\t ]+`),
	},
	Token{
		Name: ITEM_SEPERATOR,
		Shape: regexp.MustCompile(`^,`),
	},

	Token{
		Name: CALL_IN,
		Shape: regexp.MustCompile(`^\(`),

		// Create a new stack frame for the function invocation.
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			// Figure out the name of the thing that is being called.
			var callee *AstFrame
			if len(ast.Children) > 0 {
				callee = ast.Children[len(ast.Children)-1]

				// Remove the callee from the list of children.
				if len(ast.Children) == 1 {
					ast.Children = make([]*AstFrame, 0)
				} else {
					ast.Children = ast.Children[:1]
				}
			} else {
				// Can't find a callee!
				return nil, ast, ParseError(
					inp, pointer,
					"No callee identifier found before leading parenthesis in call expression.",
				)
			}

			newFrame := &AstFrame{
				Name: CALL_EXPRESSION,
				Data: map[string]interface{}{"Callee": callee},
				Parent: ast,
			}

			// Add new frame to existing AST
			ast.Children = append(ast.Children, newFrame)

			return EMPTY_DATA, newFrame, nil
		},
	},
	Token{
		Name: CALL_OUT,
		Shape: regexp.MustCompile(`^\)`),

		// Move to the previous stack frame.
		HookPostNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (*AstFrame, error) {
			var data []*AstFrame

			lastItemWasAnItemSeperator := true
			for index, child := range ast.Children {
				if child.Name == WHITESPACE { continue }
				if child.Name == ITEM_SEPERATOR {
					if lastItemWasAnItemSeperator {
						return ast, ParseError(
							inp, pointer,
							fmt.Sprintf("Two item seperators were found in a row!"),
						)
					} else {
						continue
					}
				}
				if index == 0 && child.Name == CALL_IN { continue }
				if index == len(ast.Children) - 1 && child.Name == CALL_OUT { continue }

				if child.Name == ITEM_SEPERATOR {
					lastItemWasAnItemSeperator = true
				} else {
					lastItemWasAnItemSeperator = false
					data = append(data, child)
				}
			}

			ast.Data = map[string]interface{}{ "Arguments": data }
			return ast.Parent, nil
		},
	},

	Token{
		Name: BLOCK_IN,
		Shape: regexp.MustCompile(`^do`),

		// Create a new stack frame for the function invocation.
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			newFrame := &AstFrame{
				Name: BLOCK_EXPRESSION,
				Data: EMPTY_DATA,
				Parent: ast,
			}

			// Add new frame to existing AST
			ast.Children = append(ast.Children, newFrame)

			return EMPTY_DATA, newFrame, nil
		},
	},
	Token{
		Name: BLOCK_OUT,
		Shape: regexp.MustCompile(`^end`),

		// Move to the previous stack frame.
		HookPostNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (*AstFrame, error) {
			return ast.Parent, nil
		},
	},

	Token{
		Name: ARG_LIST_IN,
		Shape: regexp.MustCompile(`^<`),

		// Create a new stack frame for the function invocation.
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			newFrame := &AstFrame{
				Name: ARG_LIST_EXPRESSION,
				Data: EMPTY_DATA,
				Parent: ast,
			}

			// Add new frame to existing AST
			ast.Children = append(ast.Children, newFrame)

			return EMPTY_DATA, newFrame, nil
		},
	},
	Token{
		Name: ARG_LIST_OUT,
		Shape: regexp.MustCompile(`^>`),

		// Move to the previous stack frame.
		HookPostNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (*AstFrame, error) {
			var data []*AstFrame

			lastItemWasAnItemSeperator := true
			for index, child := range ast.Children {
				if child.Name == WHITESPACE { continue }
				if child.Name == ITEM_SEPERATOR {
					if lastItemWasAnItemSeperator {
						return ast, ParseError(
							inp, pointer,
							fmt.Sprintf("Two item seperators were found in a row!"),
						)
					} else {
						continue
					}
				}
				if index == 0 && child.Name == ARG_LIST_IN { continue }
				if index == len(ast.Children) - 1 && child.Name == ARG_LIST_OUT { continue }

				if child.Name == ITEM_SEPERATOR {
					lastItemWasAnItemSeperator = true
				} else {
					lastItemWasAnItemSeperator = false
					data = append(data, child)
				}
			}

			ast.Data = map[string]interface{}{ "Arguments": data }
			return ast.Parent, nil
		},
	},


	// Parse different literal values
	Token{
		Name: STRING_LITERAL,
		Shape: regexp.MustCompile(`^"([^"]*)"`),

		// Add the string content inside of the data for the token
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			return map[string]interface{}{ "Content": match[1] }, ast, nil
		},
	},
	Token{
		Name: FLOAT_LITERAL,
		Shape: regexp.MustCompile(`^[0-9_]+\.[0-9_]*`),

		// Add the string content inside of the data for the token
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			float, err := strconv.ParseFloat(strings.Replace(match[0], "_", "", -1), 64)

			if err != nil {
				return EMPTY_DATA, ast, ParseError(
					inp, pointer,
					"Error parsing float from source: "+err.Error(),
				)
			}

			return map[string]interface{}{ "Content": float }, ast, nil
		},
	},
	Token{
		Name: INTEGER_LITERAL,
		Shape: regexp.MustCompile(`^[0-9_]+`),

		// Add the string content inside of the data for the token
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			integer, err := strconv.Atoi(strings.Replace(match[0], "_", "", -1))

			if err != nil {
				return EMPTY_DATA, ast, ParseError(
					inp, pointer,
					"Error parsing integer from source: "+err.Error(),
				)
			}

			return map[string]interface{}{ "Content": integer }, ast, nil
		},
	},

	Token{
		Name: IDENTIFIER,
		Shape: regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`),

		// Add the identifier name inside of the data when a new one is created.
		HookPreNew: func(
			match []string,
			ast *AstFrame,
			inp string,
			pointer int,
		) (map[string]interface{}, *AstFrame, error) {
			return map[string]interface{}{ "Name": match[0] }, ast, nil
		},
	},
}


type AstFrame struct {
	Name TokenName
	Children []*AstFrame
	Parent *AstFrame

	Data map[string]interface{}
}

// When an erro happens in the parse step, print out a pretty error, like the below:
// parse error on line 1: No valid token found!
// foo('bar')
//     ^
func ParseError(inp string, pointer int, err string) error {
	lines := strings.Split(inp, "\n")

	// Get the end of the parsed line by finding the next newline.
	endOfLineIndex := pointer + strings.Index(inp[pointer:], "\n")
	if endOfLineIndex == pointer + (-1) {
		endOfLineIndex = len(inp)
	}

	// Get the start of the parsed line by finding the previous newline.
	beginningOfLineIndex := pointer - strings.LastIndex(inp[:pointer], "\n")
	if beginningOfLineIndex == pointer - (-1) {
		beginningOfLineIndex = 0
	}

	// Get the contents of the line
	currentLine := inp[beginningOfLineIndex:endOfLineIndex]
	numberOfSpacesToPointer := pointer - beginningOfLineIndex

	// Calculate the line number now that we know the line that the error was related to.
	var lineNumber int
	for i := 0; i < len(lines); i++ {
		if lines[i] == currentLine {
			lineNumber = i+1
			break
		}
	}

	// Pad left the indicator for which the pointer points to.
	spaces := ""
	for i := 0; i < numberOfSpacesToPointer; i++ { spaces += " " }

	// Return a formatted error
	return errors.New(fmt.Sprintf(
		"parse error on line %d: %s\n%s\n%s^",
		lineNumber,
		err,
		currentLine,
		spaces,
	))
}

func Parser(inp string) (*AstFrame, error) {
	var ast AstFrame = AstFrame{Name: ROOT, Parent: nil}
	var currentFrame *AstFrame = &ast

	// Used when running hooks. Defined once and reused for each invocation.
	var err error

	// Contains the current index in `inp`
	pointer := 0

	// While items can be pulled off the front of the input...
	outer:
	for pointer < len(inp) {
		// Try to find a token that matches.
		for _, token := range TOKENS {
			if match := token.Shape.FindStringSubmatch(inp[pointer:]); len(match) > 0 {
				// Call the pre token hook to hopefully get the contents of the data to put intide
				// that ast frame.
				data := EMPTY_DATA
				if token.HookPreNew != nil {
					data, currentFrame, err = token.HookPreNew(match, currentFrame, inp, pointer)
					if err != nil { return nil, err }
				}

				// Add matching tokens to the token list, and clear the accumulator so we can find
				// the next token.
				currentFrame.Children = append(currentFrame.Children, &AstFrame{
					Name: token.Name,
					Data: data,
					Parent: currentFrame,
				})

				// Call the pre token hook to hopefully get the contents of the data to put intide
				// that ast frame.
				if token.HookPostNew != nil {
					currentFrame, err = token.HookPostNew(match, currentFrame, inp, pointer)
					if err != nil { return nil, err }
				}

				pointer += len(match[0])
				continue outer;
			}
		}

		// Found no token!
		return nil, ParseError(inp, pointer, "No valid token found!")
	}

	// Make sure that on parsing completion, we're back at the root ast node.
	if currentFrame != &ast {
		return nil, ParseError(inp, pointer, "When parsing, finished in a frame deeper than the top frame.")
	}

	return &ast, nil
}

func PrintAst(ast *AstFrame, indentation string) {
	for _, child := range ast.Children {
		fmt.Printf("%s- %s (%+v)", indentation, child.Name, child.Data)

		if value, ok := child.Data["Callee"]; ok {
			fmt.Printf("  Callee=%+v", value)
		}

		fmt.Printf("\n")
		PrintAst(child, indentation+"  ")
	}
}

func main() {
	// data := `
	// func(my_func <a b c> do
	// 	foo("bar", 123.456)
	// end)
	// `
	data := `do
		func(a<b> do 1 end)
		foo()
	end`
	// data := "foo(\"bar\")"
	ast, err := Parser(data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Ast:")
		PrintAst(ast, "")
	}
}
