package gologs

// Copied from github.com/karrick/goejs

import (
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// appendEncodedJSONFromString appends the JSON encoding of the provided string to the
// provided byte slice, and returns the modified byte slice.
//
//    func ExampleEncode() {
//        encoded := goejs.AppendEncodedJSONFromString([]byte("prefix:"), "⌘ a")
//        fmt.Printf("%s", encoded)
//        // Output: prefix:"\u0001\u2318 a"
//    }
func appendEncodedJSONFromString(buf []byte, someString string) []byte {
	buf = append(buf, '"') // prefix buffer with double quote
	for _, r := range someString {
		if escaped, ok := escapeSpecialJSON(byte(r)); ok {
			buf = append(buf, escaped...)
			continue
		}
		if r < utf8.RuneSelf && unicode.IsPrint(r) {
			buf = append(buf, byte(r))
			continue
		}
		// NOTE: Attempt to encode code point as UTF-16 surrogate pair
		r1, r2 := utf16.EncodeRune(r)
		if r1 != unicode.ReplacementChar || r2 != unicode.ReplacementChar {
			// code point does require surrogate pair, and thus two uint16 values
			buf = appendUnicodeHex(buf, uint16(r1))
			buf = appendUnicodeHex(buf, uint16(r2))
			continue
		}
		// Code Point does not require surrogate pair.
		buf = appendUnicodeHex(buf, uint16(r))
	}
	return append(buf, '"') // postfix buffer with double quote
}

// While slices in Go are never constants, we can initialize them once and reuse
// them many times. We define these slices at library load time and reuse them
// when encoding JSON.
var (
	sliceQuote          = []byte("\\\"")
	sliceBackslash      = []byte("\\\\")
	sliceSlash          = []byte("\\/")
	sliceBackspace      = []byte("\\b")
	sliceFormfeed       = []byte("\\f")
	sliceNewline        = []byte("\\n")
	sliceCarriageReturn = []byte("\\r")
	sliceTab            = []byte("\\t")
	sliceUnicode        = []byte("\\u")
)

func appendUnicodeHex(buf []byte, v uint16) []byte {
	// Start with '\u' prefix:
	buf = append(buf, sliceUnicode...)
	// And tack on 4 hexidecimal digits:
	buf = append(buf, hexDigits[(v&0xF000)>>12])
	buf = append(buf, hexDigits[(v&0xF00)>>8])
	buf = append(buf, hexDigits[(v&0xF0)>>4])
	buf = append(buf, hexDigits[(v&0xF)])
	return buf
}

const hexDigits = "0123456789ABCDEF"

// escapeSpecialJSON
func escapeSpecialJSON(b byte) ([]byte, bool) {
	// NOTE: The following 8 special JSON characters must be escaped:
	switch b {
	case '"':
		return sliceQuote, true
	case '\\':
		return sliceBackslash, true
	case '/':
		return sliceSlash, true
	case '\b':
		return sliceBackspace, true
	case '\f':
		return sliceFormfeed, true
	case '\n':
		return sliceNewline, true
	case '\r':
		return sliceCarriageReturn, true
	case '\t':
		return sliceTab, true
	}
	return nil, false
}
