package main

import (
	"fmt"
	"regexp"
	"errors"
	"strings"
)


// ----------------------------------------------------------------------------
// TOKENS
// A string is parsed into a list of tokens.
// ----------------------------------------------------------------------------
type TokenName int
const (
	IDENTIFIER TokenName = iota
	CALL_IN TokenName = iota
	CALL_OUT TokenName = iota
)

type Token struct {
	Name TokenName
	Shape *regexp.Regexp
	Data []string
}

var TOKENS = []Token{
	Token{
		Name: IDENTIFIER,
		Shape: regexp.MustCompile(`^([a-zA-Z0-9_])+`),
	},
	Token{
		Name: CALL_IN,
		Shape: regexp.MustCompile(`^\(`),
	},
	Token{
		Name: CALL_OUT,
		Shape: regexp.MustCompile(`^\)`),
	},
}


type AstFrame struct {
	Name TokenName
	Data map[string]interface{}
	Children []*AstFrame
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

func Parser(inp string) ([]Token, error) {
	var tokens []Token

	// Contains the current index in `inp`
	pointer := 0

	// While items can be pulled off the front of the input...
	for len(inp) > 0 {
		// Try to find a token that matches.
		for _, token := range TOKENS {
			if match := token.Shape.FindStringSubmatch(inp[pointer:]); len(match) > 0 {
				// Add matching tokens to the token list, and clear the accumulator so we can find
				// the next token.
				tokens = append(tokens, Token{Name: token.Name, Shape: token.Shape, Data: match})
				pointer += len(match[0])
				continue
			}
		}

		// Found no token!
		return nil, ParseError(inp, pointer, "No valid token found!")
	}

	return tokens, nil
}


func main() {
	// data := "foo('bar')\nbaz"
	data := "foo('bar')"
	tokens, err := Parser(data)
	if err != nil {
		fmt.Println("Error parsing:", err)
	} else {
		fmt.Println("Tokens:")
		fmt.Println(tokens)
	}
}
