// Copyright 2020 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package badjson is a very bad json parser. It will take almost anything.
// It respects a notation to specify byte arrays by hex or base64. See parse_test.go and the readme.
// It will parse a lot of JSON and the output from `String()` resembles JSON but it's not really
// and the objects in key:value notation are just alternating fields in a list and there's no map here.
package badjson

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Segment is a what a chunk of text will become and we'll be returning a list of them.
// Number or string or []byte are the only types.
type Segment interface {
	String() string // as json so 123 or "abc" or "=ABC" or "$414243"
	Next() Segment
	setNext(s Segment)
}

// Chop up a line of text into segments. Calling it a parser would be over stating it.
// Returns a head of a list, the number of bytes consumed, and maybe an error.
func Chop(inputLineOfText string) (Segment, error) {
	s, _, err := chop(inputLineOfText, utf8.RuneError, 0)
	return s, err
}

// closer might be } or ] when recursing
func chop(inputLineOfText string, closer rune, depth int) (Segment, int, error) {

	if len(inputLineOfText) > 16*1024 {
		return nil, 0, errors.New("is longer than 16k")
	}
	if depth >= 16 {
		return nil, 0, errors.New("recursed 16 deep")
	}
	// we're returning a linked list, head.Next() which may be nil
	var head base
	var tail Segment
	tail = &head

	// working variables for scanning loop below
	str := inputLineOfText[:]
	i := 0
	start := i
	r, runeLength := utf8.DecodeRuneInString(str[i:])

	// define some terms we'll need.
	done := func() bool { // return true if done
		return i >= len(str) || runeLength == 0
	}
	pop := func() bool { // advance and return true if done
		i += runeLength
		if i < len(str) {
			r, runeLength = utf8.DecodeRuneInString(str[i:])
		} else {
			r = closer
			return true
		}
		return done()
	}
	currentString := func() string {
		return str[start:i]
	}
	isHex := func() bool { // is r is a char used in hexadecimal encoding?
		return runeLength == 1 && HexMap[r] != byte(0xFF)
	}
	isB64 := func() bool { // is r is a char used in base64 encoding?
		return runeLength == 1 && B64DecodeMap[r] != byte(0xFF)
	}

	for { // while not done() scan the input text.
		// pass delimeters
		for r == ' ' {
			if pop() {
				return head.nexts, i, nil
			}
		}
		if r == ',' || r == ':' {
			if pop() {
				return head.nexts, i, nil
			}
		}
		for r == ' ' {
			if pop() {
				return head.nexts, i, nil
			}
		}
		start = i
		if r == closer {
			return head.Next(), i, nil
		}
		switch r {
		case '$': // hex for a byte array
			if pop() {
				start = i
				goto donehexarray // output empty array
			}
			start = i
			for isHex() {
				if pop() {
					break
				}
			}
		donehexarray:
			tail = NewHexBytes(currentString(), tail)
		case '"', 39: // 39 is ' // quoted string, unescaped later
			quote := r
			if pop() {
				break
			}
			start = i
			hasEscape := false
			for r != quote {
				if r == '\\' {
					hasEscape = true
					if pop() {
						break
					}
				}
				if pop() {
					break
				}
			}
			tail = NewRuneArray(currentString(), tail, hasEscape)
			if pop() {
				break
			}
		case '+', '-': // numbers
			sign := r
			if pop() { // pass the +
				break
			}
			start = i
			previousr := r
		morenum:
			for r != ' ' && r != ':' && r != ',' && r != '+' && r != '-' && r != closer {
				previousr = r
				if pop() {
					break
				}
			}
			if (r == '+' || r == '-') && previousr == 'e' {
				if pop() {
				} else {
					goto morenum
				}
			}
			tail = NewNumber(currentString(), tail, sign == '-')
		case '=':
			var sss string
			if pop() { // pass the =
				sss = currentString()
				goto doneb64array // output empty array
			}
			start = i
			for isB64() {
				if pop() {
					break
				}
			}
			sss = currentString()
			for r == '=' { // pass any ='s at the end
				if pop() {
					break
				}
			}
		doneb64array:
			tail = NewBase64Bytes(sss, tail)
		case '{', '[':
			paren := r
			if pop() {
				break
			}
			start = i
			closewith := ']'
			if paren == '{' {
				closewith = '}'
			}
			childList, newi, err := chop(str[i:], closewith, depth+1)
			if err != nil {
				return head.Next(), newi, nil
			}
			i = i + newi
			tail = NewParent(tail, childList, paren == '[')
			if pop() {
				break
			}
		default:
			// an unquoted string
			for r != ' ' && r != ':' && r != ',' && r != closer {
				if pop() {
					break
				}
			}
			tail = NewRuneArray(currentString(), tail, false)
		}
	}
}

// ToString will wrap the list with `[` and `]` and output like child list.
func ToString(segment Segment) string {
	var sb strings.Builder
	sb, _ = getJSONinternal(segment, sb, true)
	result := sb.String()
	return result
}

//  expresses a list of Segment's as JSON, Is the String() of the Parent object.
func getJSONinternal(s Segment, dest strings.Builder, isArray bool) (strings.Builder, error) {

	oddDelimeter := ','
	if isArray {
		dest.WriteByte('[')
	} else {
		dest.WriteByte('{')
		oddDelimeter = ':'
	}

	for i := 0; s != nil; s = s.Next() {
		if i != 0 {
			if i&1 != 1 {
				dest.WriteRune(',')
			} else {
				dest.WriteRune(oddDelimeter)
			}
		}
		dest.WriteString(s.String())
		i++
	}
	if isArray {
		dest.WriteByte(']')
	} else {
		dest.WriteByte('}')
	}

	_ = s

	return dest, nil
}

// they will all decend from base
type base struct {
	nexts Segment
	input string
}

func (b *base) String() string {
	return `""`
}

func (b *base) Next() Segment {
	return b.nexts
}

func (b *base) setNext(n Segment) {
	b.nexts = n
}

// Base64Bytes for when there's a block of data in base64
type Base64Bytes struct {
	base
	output []byte
}

// NewBase64Bytes is a factory
func NewBase64Bytes(data string, previous Segment) Segment {
	b := new(Base64Bytes)
	b.input = data
	previous.setNext(b)
	return b
}

// GetBytes try to parse
func (b *Base64Bytes) GetBytes() []byte {
	decoded, err := base64.RawStdEncoding.DecodeString(b.input)
	if err != nil {
		//fmt.Println("decode error:", err)
		return []byte("")
	}
	return decoded
}

func (b *Base64Bytes) String() string {
	bytes := b.GetBytes()
	str := base64.RawStdEncoding.EncodeToString(bytes)
	return `"=` + str + `"`
}

// HexBytes is for when there's a block of data in hex.
type HexBytes struct {
	base
}

// NewHexBytes is a factory
func NewHexBytes(data string, previous Segment) Segment {
	b := new(HexBytes)
	b.input = data
	previous.setNext(b)
	return b
}

// GetBytes try to parse
func (b *HexBytes) GetBytes() []byte {
	in := b.input
	if len(in)&1 == 1 {
		in = in + "0"
	}
	decoded, err := hex.DecodeString(in)
	// it's impossible to get test coberage for this:
	// if err != nil {
	// 	fmt.Println("decode error:", err)
	// 	return []byte("")
	// }
	_ = err
	return decoded
}

func (b *HexBytes) String() string {
	bytes := b.GetBytes()
	encodedStr := hex.EncodeToString(bytes)
	return `"$` + encodedStr + `"`
}

// RuneArray aka string
type RuneArray struct {
	base
	hasEscape bool
}

// NewRuneArray is a factory
func NewRuneArray(data string, previous Segment, hasEscape bool) Segment {
	b := new(RuneArray)
	b.input = data
	previous.setNext(b)
	b.hasEscape = hasEscape
	return b
}

// GetString to return the unescaped string
func (b *RuneArray) GetString() string {
	if b.hasEscape {
		var sb strings.Builder
		sb.Grow(len(b.input))
		passed := false
		for _, r := range b.input {
			if r == '\\' && !passed {
				passed = true
			} else {
				sb.WriteRune(r)
				passed = false
			}
		}
		return sb.String()
	}
	return b.input
}

func needsEscape(str string) int {
	count := 0
	for _, r := range str {
		if r == '\\' || r == '\'' || r == '"' {
			count++
		}
	}
	return count
}

func makeEscaped(str string, sizeHint int) string {
	var sb strings.Builder
	sb.Grow(len(str) + sizeHint)
	for _, r := range str {
		if r == '\\' || r == '\'' || r == '"' {
			sb.WriteRune('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// Return the string in json format
func (b *RuneArray) String() string {
	str := b.GetString()
	needAmt := needsEscape(str)
	if needAmt > 0 {
		str = makeEscaped(str, needAmt)
	}
	return `"` + str + `"`
}

// Number is a float64
type Number struct {
	base
	wasNegative bool
}

// NewNumber is a factory
func NewNumber(data string, previous Segment, wasNegative bool) Segment {
	b := new(Number)
	b.input = data
	b.wasNegative = wasNegative
	previous.setNext(b)
	return b
}

// GetNumber parses errors into zeros.
func (b *Number) GetNumber() float64 {
	var val float64
	if len(b.input) == 0 {
		return val
	}
	if b.input[0] == '$' {
		if len(b.input) >= 2 {
			ival, _ := strconv.ParseInt(b.input[1:], 16, 64)
			val = float64(ival)
		} else {
			val = 0
		}
	} else {
		val, _ = strconv.ParseFloat(b.input, 64)
	}
	if b.wasNegative {
		val = -val
	}
	return val
}

func (b *Number) String() string {
	val := b.GetNumber()
	prefix := ""
	if val >= 0 {
		prefix = "+"
	}
	if float64(int64(val)) == val {
		return prefix + strconv.FormatInt(int64(val), 10)
	}
	return prefix + strconv.FormatFloat(val, 'g', -1, 64)
}

// Parent has a sub-list
type Parent struct {
	base
	children Segment
	wasArray bool
}

// NewParent is a factory
func NewParent(previous Segment, children Segment, wasArray bool) Segment {
	b := new(Parent)
	previous.setNext(b)
	b.children = children
	b.wasArray = wasArray
	return b
}

func (b *Parent) String() string {
	var sb strings.Builder
	sb, _ = getJSONinternal(b.children, sb, b.wasArray)
	return sb.String()
}

// Dummy is dumb
func Dummy() {
	// and stupid
}

const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// B64DecodeMap from ascii to b64
var B64DecodeMap [256]byte

// HexMap has values for hex
var HexMap [256]byte

const isHex = "0123456789ABCDEFabcdef"

func init() {

	for i := 0; i < len(B64DecodeMap); i++ {
		B64DecodeMap[i] = byte(0xFF)
	}
	for i := 0; i < len(HexMap); i++ {
		HexMap[i] = byte(0xFF)
	}
	for i := 0; i < len(encodeStd); i++ {
		r := encodeStd[i]
		B64DecodeMap[r] = byte(i)
	}
	for i := 0; i <= 10; i++ {
		HexMap['0'+i] = byte(i)
	}
	for i := 10; i <= 16; i++ {
		HexMap['a'+i-10] = byte(i)
		HexMap['A'+i-10] = byte(i)
	}

}
