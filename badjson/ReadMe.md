## BadJSON text recognizer.

The purpose is to divide up a line of text into byte arrays and numbers. I'll call those Segments. Think of a command line becoming the args to a function. In particular I needed a utility so I can paste binary keys into a command line. It's a fun game to try to write that concisely and elegantly but also common and readable. 

Whitespace is skipped and then if the next character after the whitespace is `"` then succeeding utf-8 characters are collected into a string. 

For instance: The input `"astring"` is parsed as the byte array of the utf-8 of `astring`. Single quotes, `'`, are also used and there is escaping with `\`.

If the first utf-8 glyph after the whitespace is `=` then following characters are collected that are Base64 encoding compatible. Eg `  =xyzw  ` is parsed as 3 bytes `c7 2c f0`

Likewise ff the first utf-8 glyph after the whitespace is `$` then following characters are collected that are Base64 encoding compatible. Eg `$c72cf0` is parsed as 3 bytes `c7 2c f0`.

The `+` and `-` are used as the start of numbers. 

The characters `{` and `[` are used to form sub-lists of Segments.

The characters `,` and `:` are used as delimeters between segments besides spaces. Any segments that are unrecognized are collected as strings.

The code is here and is just one file and can be included as `"github.com/awootton/knotfreeiot/badjson"` for anyone who wants to `import` a toy parser. 

In the interests of simplicity all instances of malformed input are interpreted in a default way instead of enumerating the error conditions. Bad number inputs return zero and bad byte array syntax returns empty errays. 

The `String()` methods of the types produced returns in a JSON compatible way. The generality of the recognition means that JSON will parse but the output is not fully compatible as `+` precedes numbers.

Examples:  `abc:def,ghi:jkl` parses into `["abc","def","ghi","jkl"]`.

`"abc":"def","ghi":"jkl"` parses into `["abc","def","ghi","jkl"]`



