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

package badjson_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/awootton/knotfreeiot/badjson"
)

/** Notes. Parse string into a linked list of segments.

$ starts hex array of bytes

" and ' start string with escape character `\`.

- and + start decimal number with -$ and +$ for hex

space and : and , are separators.

= starts base64 data

{ and [ start a recursion.

Everything else is a utf string.

Code coverage is 100%.

*/

func ExampleChop() {

	someText := `abc:def,ghi:jkl`

	// parse the text
	segment, err := badjson.Chop(someText)
	if err != nil {
		fmt.Println(err)
	}
	// traverse the result
	for s := segment; s != nil; s = s.Next() {
		fmt.Println(reflect.TypeOf(s))
	}
	// output it
	output := badjson.ToString(segment)
	fmt.Println(output)

	someText = `"abc""def""ghi""jkl"`
	segment, err = badjson.Chop(someText)
	if err != nil {
		fmt.Println(err)
	}
	output = badjson.ToString(segment)
	fmt.Println(output)

	// Expect: *badjson.RuneArray
	// *badjson.RuneArray
	// *badjson.RuneArray
	// *badjson.RuneArray
	// ["abc","def","ghi","jkl"]
	// ["abc","def","ghi","jkl"]
}

func binaryTests() {

	fmt.Println("abc", hex.EncodeToString([]byte("abc")))
	fmt.Println("def", hex.EncodeToString([]byte("def")))
	fmt.Println("ghi", hex.EncodeToString([]byte("ghi")))
	fmt.Println("jkl", hex.EncodeToString([]byte("jkl")))

	fmt.Println("abc", base64.RawURLEncoding.EncodeToString([]byte("abc")))
	fmt.Println("def", base64.RawURLEncoding.EncodeToString([]byte("def")))
	fmt.Println("ghi", base64.RawURLEncoding.EncodeToString([]byte("ghi")))
	fmt.Println("jkl", base64.RawURLEncoding.EncodeToString([]byte("jkl")))

}

func TestExample(t *testing.T) {
	binaryTests()
	ExampleChop()
}

func TestParse1(t *testing.T) {

	got := "a"
	want := "b"

	got = getOneString(`"ab\cd"`)
	want = `abcd`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(`"ab\cd"`)
	want = `["abcd"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = getOneString(`"ab\'cd"`)
	want = `ab'cd`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(`"ab\'cd"`)
	want = `["ab\'cd"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = getOneString(`"ab\"cd"`)
	want = `ab"cd`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(`"ab\"cd"`)
	want = `["ab\"cd"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = getOneString(`"ab\\cd"`)
	want = `ab\cd`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(`"ab\\cd"`)
	want = `["ab\\cd"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = getOneString(`    ab\"cd   `)
	backslash := rune(92)
	quote := rune(34)
	if rune([]byte(got)[2]) != backslash {
		t.Errorf("got %v, want %v", []byte(got)[2], backslash)
	}
	if rune([]byte(got)[3]) != quote {
		t.Errorf("got %v, want %v", []byte(got)[2], backslash)
	}
	want = `ab\"cd`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(`ab\"cd`)
	want = `["ab\\\"cd"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func xxTestParse2(t *testing.T) {

	got := "a"
	want := "b"

	got = tryParseAndUnparse("+2e-2")
	want = `[+0.02]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" +1234567 ")
	want = `[+1234567]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" +11e2 +.0001  +2e-2")
	want = `[+1100,+0.0001,+0.02]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParse3(t *testing.T) {

	got := "a"
	want := "b"

	got = tryParseAndUnparse("{a b}}")
	want = `[{"a":"b"},"}"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse("[a b]]")
	want = `[["a","b"],"]"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse("{{{{{{{{{{{{{{{{a b}}}}}}}}}}}}}}}}")
	want = `[{{{{{{{{{{{{{{{}:"a","b"}}}}}}}}}}}}}},"}}"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse("{{{{{{{{{{{{{{{{{{a b}")
	// note that it refuses to recurse that deep so the last '{'
	// becomes a sibling and not a child. bad parser. bad.
	want = `[{{{{{{{{{{{{{{{}:{},"a":"b"}}}}}}}}}}}}}}]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestParse4(t *testing.T) {

	got := "a"
	want := "b"

	// got = tryParseAndUnparse(" $1234 $45678 +$1234 -$8765")
	// want = `["$1234","$456780",+4660,-34661]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	got = tryParseAndUnparse("   aaa : bbb ")
	want = `["aaa","bbb"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse("{{ osiojdhnd : hhh44 [[[ }    ")
	want = `[{{"osiojdhnd":"hhh44",[[["}"]]]}}]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" {a:b,c:d}  ")
	want = `[{"a":"b","c":"d"}]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" a ")
	want = `["a"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" a")
	want = `["a"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse("abc ")
	want = `["abc"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// got = tryParseAndUnparse("a +1234 MyName_var " + `"quoted string"` + " 'another' " + "=isuhe48r8dhbsvs  ")
	// want = `["a",+1234,"MyName_var","quoted string","another","=isuhe48r8dhbsvs"]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	got = tryParseAndUnparse("[      []]   ")
	want = `[[]]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// exponents and fractions are zeros. ++ is two zeros.
	// got = tryParseAndUnparse("+1+2+3-4 ++++ ---- ++$+ --$- +1e4 +7e-4 fails!! +.00007 is_ok ")
	// want = `[+1,+2,+3,-4,+0,+0,+0,+0,+0,+0,+0,+0,+0,+0,+0,+0,+0,+0,+10000,+0.0007,"fails!!",+7e-05,"is_ok"]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }
}

func getOneString(input string) string {
	segment, err := badjson.Chop(input)
	ra, ok := segment.(*badjson.RuneArray)
	if !ok {
		return "this is not the string you were looking for " + err.Error()
	}
	return ra.GetString()
}

// check for zombies
func TestParseZ(t *testing.T) {

	got := "a"
	want := "b"
	var sb strings.Builder
	for i := 0; i < 1024; i++ {
		sb.WriteString("0123456789abcdef")
	}
	got = getOneString(sb.String())
	want = sb.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want[0:100])
	}
	sb.WriteString("a")
	got = getOneString(sb.String())
	want = `this is not the string you were looking for is longer than 16k`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" ,")
	want = `[]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" , ")
	want = `[]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" aaa $")
	want = `["aaa","$"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(" aaa $F")
	want = `["aaa","$f0"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` aaa "`)
	want = `["aaa"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` aaa "\`)
	want = `["aaa",""]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` "unterminated`)
	want = `["unterminated"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// got = tryParseAndUnparse(` +`)
	// want = `[]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }
	// got = tryParseAndUnparse(` +1`)
	// want = `[+1]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }
	// got = tryParseAndUnparse(` +1e`)
	// want = `[+0]` // because it's a float parse error
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	// got = tryParseAndUnparse(` +1e+`)
	// want = `[+0]` // because it's a float parse error
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	got = tryParseAndUnparse(` =`)
	want = `["="]` // because empty
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` =ABC`)
	want = `["=ABA"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` $`) // ends before hex
	want = `["$"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(` $smelly  `) // ends before hex
	want = `["$","smelly"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(` $  `) // ends before hex
	want = `["$"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(` =a==  `) // base64 parse error
	want = `["="]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(` =a=`) // base64 parse error
	want = `["="]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = tryParseAndUnparse(` =aa=`)
	want = `["=aQ"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = tryParseAndUnparse(`{`)
	want = `[]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	abase := badjson.Base{}
	got = abase.String()
	want = `""`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func tryParseAndUnparse(str string) string {

	segment, err := badjson.Chop(str)
	if err != nil {
		return "ERROR_" + err.Error()
	}

	result := badjson.ToString(segment)

	segment2, _ := badjson.Chop(result[1 : len(result)-1])

	result2 := badjson.ToString(segment2)

	if result != result2 {
		return result + "!=" + result2
	}
	return result
}
