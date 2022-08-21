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

// Package comments. You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package badjson is a very bad json parser. It will take almost anything.
// It respects a notation to specify byte arrays by hex or base64. See parse_test.go and the readme.
// It will parse a lot of JSON and the output from `String()` resembles JSON but it's not really
// and the objects in key:value notation are just alternating fields in a list and there's no map here.
// 3/2020 Commented out all the number recognitions since we're not using it.
package badjson

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

// Segment is a what a chunk of text will become and we'll be returning a list of them.
// Number or string or []byte are the only types.
type Segment interface {
	Next() Segment
	setNext(s Segment)
	GetQuoted() string // as json so 123 or "abc" or "=ABC" or "$414243"
	Raw() string       // unquoted
}

type Base struct {
	nexts Segment
	input string
}

// RuneArray is a span of runes with quoting hints
type RuneArray struct {
	Base
	theQuote        int // is 0 or "" or '
	hadQuoteOrSlash bool
}

// Chop up a line of text into segments. Calling it a parser would be overstating.
// Returns a head of a list, the number of bytes consumed, and maybe an error.
func Chop(inputLineOfText string) (Segment, error) {
	s, _, err := chop(inputLineOfText, utf8.RuneError, 0)
	if err != nil {
		return s, err
	}
	if s == nil { // parsed nothing
		//not fond of returning nil return s, errors.New("no content")
		b := RuneArray{}
		b.input = ""
		s = &b
	} else if s.Next() == nil { // special case for '[ content ]' just return content
		parent := AsParent(s)
		// if has child, and was an array
		// return child
		if parent != nil && parent.wasArray && parent.children != nil {
			return parent.children, nil
		}
	}
	return s, nil
}

// TODO: rewrite without recursion (stack the head and the tail).
// closer might be } or ] when recursing
// it returns the head segment, a count of the chars used used, and a possible error.
func chop(inputLineOfText string, closer rune, depth int) (Segment, int, error) {

	if len(inputLineOfText) > 16*1024 {
		return nil, 0, errors.New("is longer than 16k")
	}
	if depth > 16 {
		return nil, 0, errors.New("recursed 16 deep")

	}
	var front Segment // the first element of the linked list
	var tail Segment  // the last

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
	linktoTail := func(s Segment) {
		if front == nil {
			front = s
		}
		if tail != nil {
			tail.setNext(s)
		}
		tail = s
	}

	for { // while not done() scan the input text.
		// pass delimeters
		for r == ' ' {
			if pop() {
				return front, i, nil
			}
		}
		if r == ',' || r == ':' {
			if pop() {
				return front, i, nil
			}
		}
		for r == ' ' {
			if pop() {
				return front, i, nil
			}
		}
		start = i
		if r == closer {
			return front, i, nil
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
			hb := new(HexBytes)
			hb.input = currentString()
			linktoTail(hb)
		case '"', 39: // 39 is ' // quoted string, unescaped later
			quote := r
			if pop() {
				break
			}
			start = i
			hadQuoteOrSlash := false
			for r != quote {
				if r == '\\' {
					hadQuoteOrSlash = true
					if pop() {
						break
					}
				} else if r == '"' {
					hadQuoteOrSlash = true
				}
				if pop() {
					break
				}
			}
			//needsQuote := true
			ra := new(RuneArray)
			ra.hadQuoteOrSlash = hadQuoteOrSlash
			ra.theQuote = int(quote)
			ra.input = currentString()
			linktoTail(ra)
			if pop() {
				break
			}
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
			ba64 := new(Base64Bytes) //(sss, tail)
			ba64.input = sss
			ba64.input = currentString()
			linktoTail(ba64)
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
				return front, i + newi, err
			}
			i = i + newi
			par := new(Parent)
			par.children = childList
			par.wasArray = paren == '['
			linktoTail(par)
			if i >= len(str) {
				return front, len(str), nil
			}
			if pop() {
				break
			}
		default:
			// an unquoted string
			hadQuoteOrSlash := false
			for r != ' ' && r != ':' && r != ',' && r != closer {

				if r == '"' || r == '\\' {
					hadQuoteOrSlash = true
				}
				if pop() {
					break
				}
			}
			ra := new(RuneArray)
			ra.hadQuoteOrSlash = hadQuoteOrSlash
			ra.theQuote = 0
			ra.input = currentString()
			linktoTail(ra)
		}
	}
}

// ToString will wrap the list with `[` and `]` and output like child list.
// todo: move to testing.
func ToString(segment Segment) string {
	var sb strings.Builder
	sb, _ = getJSONinternal(segment, sb, true)
	result := sb.String()
	return result
}

// expresses a list of Segment's as JSON, Is the GetQuoted() of the Parent object.
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
		dest.WriteString(s.GetQuoted())
		i++
	}
	if isArray {
		dest.WriteByte(']')
	} else {
		dest.WriteByte('}')
	}
	return dest, nil
}

// Next returns the next segment or nil
func (b *Base) Next() Segment {
	return b.nexts
}
func (b *Base) setNext(n Segment) {
	b.nexts = n
}

// Base64Bytes for when there's a block of data in base64
type Base64Bytes struct {
	Base
}

// GetBytes try to parse b64 to bytes
func (b *Base64Bytes) GetBytes() []byte {
	decoded, err := base64.RawURLEncoding.DecodeString(b.input)
	if err != nil {
		return []byte("")
	}
	return decoded
}

func (b *Base64Bytes) GetQuoted() string {
	return `"` + b.Raw() + `"`
}

// Raw decodes and then reencodes because the input can be weird
func (b *Base64Bytes) Raw() string {
	bytes := b.GetBytes()                              // why do we decode
	str := base64.RawURLEncoding.EncodeToString(bytes) // and then encode?
	return `=` + str + ``
}

// HexBytes is for when there's a block of data in hex.
type HexBytes struct {
	Base
}

// GetBytes try to parse
func (b *HexBytes) GetBytes() []byte {
	in := b.input
	if len(in)&1 == 1 {
		in = in + "0"
	}
	decoded, err := hex.DecodeString(in)
	// it's impossible to get test coverage for this:
	// if err != nil {
	// 	fmt.Println("decode error:", err)
	// 	return []byte("")
	// }
	_ = err
	return decoded
}

func (b *HexBytes) GetQuoted() string {
	return `"` + b.Raw() + `"`
}

// Raw is unquoted
func (b *HexBytes) Raw() string {
	bytes := b.GetBytes()                   // why do we decode
	encodedStr := hex.EncodeToString(bytes) // and then encode? fixeme:
	return `$` + encodedStr + ``
}

// MakeEscaped will return an 'escaped' version of the string when string contains \ or "
// the usual escaping for json values and keys
func MakeEscaped(str string) string {
	var sb strings.Builder
	sb.Grow(len(str))

	for _, r := range str {
		if r == '\\' || r == '"' { // we can  || r == '\''
			sb.WriteRune('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// MakeUnescaped if we find a \ followed by a \ or a " then skip it
func MakeUnescaped(str string, theQuote rune) string {
	var sb strings.Builder
	sb.Grow(len(str))

	passedSlash := false
	for _, r := range str {
		if r == '\\' && !passedSlash {
			// don't output just yet
			passedSlash = true
		} else {
			if passedSlash && r != '\\' && r != theQuote {
				// we needed to output that slash after all
				sb.WriteRune('\\')
			}
			sb.WriteRune(r)
			passedSlash = false
		}
	}
	return sb.String()
}

// Return the string in json format
// so we always quote with " and never '
func (b *RuneArray) GetQuoted() string {

	var sb strings.Builder
	sb.Grow(len(b.input))

	sb.WriteRune('"')

	if b.theQuote == int('"') {
		// it was quoted
		if b.hadQuoteOrSlash {

			sb.WriteString(b.input)

		} else {

			sb.WriteString(MakeEscaped(b.input))
		}
	} else if b.theQuote == '\'' {

		tmp := MakeUnescaped(b.input, '\'')
		tmp = MakeEscaped(tmp)
		sb.WriteString(tmp)

	} else {
		// wasn't quoted.
		sb.WriteString(MakeEscaped(b.input))
	}
	sb.WriteRune('"')

	return sb.String()
}

// Raw returns the 'original' string with no escaping
func (b *RuneArray) Raw() string {

	if b.theQuote == int('"') {
		// it was quoted
		if b.hadQuoteOrSlash {
			// we'll have to un escape it.
			return MakeUnescaped(b.input, '"')

		} else {
			return b.input
		}
	} else if b.theQuote == '\'' {
		// it was quoted but with ' and not "
		if b.hadQuoteOrSlash {
			// we'll have to un escape it.
			return MakeUnescaped(b.input, '\'')

		} else {
			return b.input
		}
	} else {
		// wasn't quoted.
		return b.input
	}
}

// Parent has a sub-list
type Parent struct {
	Base
	children Segment
	wasArray bool
}

// NewParent is a factory
func xxNewParent(previous Segment, children Segment, wasArray bool) Segment {
	b := new(Parent)
	//previous.setNext(b)
	b.children = children
	b.wasArray = wasArray
	return b
}

func (b *Parent) GetQuoted() string {
	var sb strings.Builder
	sb, _ = getJSONinternal(b.children, sb, b.wasArray)
	return sb.String()
}

// Raw is
func (b *Parent) Raw() string {
	var sb strings.Builder
	sb, _ = getJSONinternal(b.children, sb, b.wasArray)
	return sb.String()
}

// const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
// No. Use the url version:
const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/-_"

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

// IsASCII is true if all chars are >= ' ' and <= 127
// the 2nd bool is if the string has delimeters so it would *need quotes*.
func IsASCII(bstr []byte) (bool, bool) {
	isascii := true
	hasdelimeters := false
	r, runeLength := utf8.DecodeRune(bstr)
	if runeLength == 1 {
		//if r == '"' || r == ',' || r == ':' || r == ' ' || r == '$' || r == '+' || r == '-' || r == '=' || r == '[' || r == '{' {
		if r == '"' || r == ',' || r == ':' || r == ' ' || r == '$' || r == '=' || r == '[' || r == '{' {
			hasdelimeters = true
		}
	}

	for _, b := range bstr {
		if b < 32 || b > 127 {
			isascii = false

		} else {
			if b == ' ' || b == ':' || b == ',' { // ] and } ?
				hasdelimeters = true
			}
		}
	}

	return isascii, hasdelimeters
}

// AsParent returns pointer to Parent if s is a Parent
func AsParent(s Segment) *Parent {
	ch, ok := s.(*Parent)
	if ok {
		return ch
	}
	return nil
}
