package main

import (
	"bufio"
	"regexp"
	"strconv"
	"unicode/utf8"
)

// Lexer

// Lexer returns tokens, either [0-utf8.MaxRune] for
// unknown runes, or one of these for known things.
const (
	tEof = -1 - iota
	tDef
	tExtern
	tIdentifier
	tNumber
)

var (
	identifierName string  // tIdentifier's name if last call to getToken returned tIdentifier
	numberValue    float64 // the last tNumber's value
	scanner        *bufio.Scanner
	isWord         = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	isNumber       = regexp.MustCompile(`^[0-9.]+`)
)

func getToken() rune {
	var err error
	if !scanner.Scan() {
		return tEof
	}

	chars := scanner.Bytes()

	if isWord.Match(chars) {
		identifierName = scanner.Text()
		switch identifierName {
		case "def":
			return tDef
		case "extern":
			return tExtern
		default:
			return tIdentifier
		}
	}

	if isNumber.Match(chars) {
		numberValue, err = strconv.ParseFloat(scanner.Text(), 64)
		check(err)
		return tNumber
	}

	if scanner.Text() == "/*" {
		for scanner.Text() != "*/" {
			if !scanner.Scan() {
				return tEof
			}
		}
	}

	r, _ := utf8.DecodeRune(chars)
	return r
}
