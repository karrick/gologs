package gologs

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func ensureError2(tb testing.TB, testCase string, err error, contains string) {
	tb.Helper()
	if err == nil || !strings.Contains(err.Error(), contains) {
		tb.Errorf("Case: %#q; GOT: %v; WANT: %v", testCase, err, contains)
	}
}
func stringEnsureBad(tb testing.TB, input, errorMessage, remainder string) {
	tb.Helper()
	_, buf2, err := decodedStringFromJSON([]byte(input))
	ensureError2(tb, input, err, errorMessage)
	if actual, expected := string(buf2), remainder; actual != expected {
		tb.Errorf("Input: %#q; Remainder GOT: %#q; WANT: %#q; Error: %s", input, actual, expected, err)
	}
}

func stringEnsureGood(tb testing.TB, input, expected string) {
	tb.Helper()
	buf := appendEncodedJSONFromString(nil, input)
	if actual := string(buf); actual != expected {
		tb.Errorf("Input: %#q; GOT: %#q; WANT: %#q", input, actual, expected)
	}
	output, buf2, err := decodedStringFromJSON([]byte(expected))
	if err != nil {
		tb.Errorf("Input: %#q: %s", input, err)
	}
	if input != output {
		tb.Errorf("Input: %#q; Output: %#q", input, output)
	}
	if actual, expected := string(buf2), ""; actual != expected {
		tb.Errorf("Input: %#q; Remainder GOT: %#q; WANT: %#q", input, actual, expected)
	}
}

func ExampleStringDecode() {
	decoded, remainder, err := decodedStringFromJSON([]byte("\"\\u0001\\u2318 a\" some extra bytes after final quote"))
	if err != nil {
		fmt.Println(err)
	}
	if actual, expected := string(remainder), " some extra bytes after final quote"; actual != expected {
		fmt.Printf("Remainder GOT: %#q; WANT: %#q\n", actual, expected)
	}
	fmt.Printf("%#q", decoded)
	// Output: "\x01⌘ a"
}

func ExampleStringEncode() {
	encoded := appendEncodedJSONFromString([]byte("prefix:"), "⌘ a")
	fmt.Printf("%s", encoded)
	// Output: prefix:"\u0001\u2318 a"
}

func TestString(t *testing.T) {
	stringEnsureBad(t, `"`, "short buffer", "\"")
	stringEnsureBad(t, `..`, "expected initial '\"'", "..")
	stringEnsureBad(t, `".`, "expected final '\"'", "\".")

	stringEnsureGood(t, "", "\"\"")
	stringEnsureGood(t, "a", "\"a\"")
	stringEnsureGood(t, "ab", "\"ab\"")
	stringEnsureGood(t, "a\"b", "\"a\\\"b\"")
	stringEnsureGood(t, "a\\b", "\"a\\\\b\"")
	stringEnsureGood(t, "a/b", "\"a/b\"")

	stringEnsureGood(t, "a\bb", `"a\bb"`)
	stringEnsureGood(t, "a\fb", `"a\fb"`)
	stringEnsureGood(t, "a\nb", `"a\nb"`)
	stringEnsureGood(t, "a\rb", `"a\rb"`)
	stringEnsureGood(t, "a\tb", `"a\tb"`)
	stringEnsureGood(t, "a	b", `"a\tb"`) // tab byte between a and b

	stringEnsureBad(t, "\"\\u\"", "short buffer", "\"")
	stringEnsureBad(t, "\"\\u.\"", "short buffer", ".\"")
	stringEnsureBad(t, "\"\\u..\"", "short buffer", "..\"")
	stringEnsureBad(t, "\"\\u...\"", "short buffer", "...\"")

	stringEnsureBad(t, "\"\\u////\"", "invalid byte", "////\"") // < '0'
	stringEnsureBad(t, "\"\\u::::\"", "invalid byte", "::::\"") // > '9'
	stringEnsureBad(t, "\"\\u@@@@\"", "invalid byte", "@@@@\"") // < 'A'
	stringEnsureBad(t, "\"\\uGGGG\"", "invalid byte", "GGGG\"") // > 'F'
	stringEnsureBad(t, "\"\\u````\"", "invalid byte", "````\"") // < 'a'
	stringEnsureBad(t, "\"\\ugggg\"", "invalid byte", "gggg\"") // > 'f'

	stringEnsureGood(t, "⌘ ", "\"\\u0001\\u2318 \"")
	stringEnsureGood(t, "😂 ", "\"\\u0001\\uD83D\\uDE02 \"")
	stringEnsureGood(t, `☺️`, `"\u263A\uFE0F"`)
	stringEnsureGood(t, `日本語`, `"\u65E5\u672C\u8A9E"`)

	stringEnsureBad(t, "\"\\uD83D\"", "surrogate pair", "")
	stringEnsureBad(t, "\"\\uD83D\\u\"", "surrogate pair", "u\"")
	stringEnsureBad(t, "\"\\uD83D\\uD\"", "surrogate pair", "uD\"")
	stringEnsureBad(t, "\"\\uD83D\\uDE\"", "surrogate pair", "uDE\"")
	stringEnsureBad(t, "\"\\uD83D\\uDE0\"", "invalid byte", "uDE0\"")
}

// decodedStringFromJSON decodes a string from JSON, returning the decoded
// string and the remainder byte slice of the original buffer. On error, the
// returned byte slice points to the first byte that caused the error indicated.
//
//	func ExampleDecode() {
//	    decoded, remainder, err := goejs.DecodedStringFromJSON([]byte("\"\\u0001\\u2318 a\" some extra bytes after final quote"))
//	    if err != nil {
//	        fmt.Println(err)
//	    }
//	    if actual, expected := string(remainder), " some extra bytes after final quote"; actual != expected {
//	        fmt.Printf("Remainder GOT: %#q; WANT: %#q\n", actual, expected)
//	    }
//	    fmt.Printf("%#q", decoded)
//	    // Output: "\x01⌘ a"
//	}
func decodedStringFromJSON(buf []byte) (string, []byte, error) {
	buflen := len(buf)
	if buflen < 2 {
		return "", buf, fmt.Errorf("cannot decode string: %s", io.ErrShortBuffer)
	}
	if buf[0] != '"' {
		return "", buf, fmt.Errorf("cannot decode string: expected initial '\"'; found: %#U", buf[0])
	}
	var newBytes []byte
	var escaped, ok bool
	// Loop through bytes following initial double quote, but note we will
	// return immediately when find unescaped double quote.
	for i := 1; i < buflen; i++ {
		b := buf[i]
		if escaped {
			escaped = false
			if b, ok = unescapeSpecialJSON(b); ok {
				newBytes = append(newBytes, b)
				continue
			}
			if b == 'u' {
				// NOTE: Need at least 4 more bytes to read uint16, but subtract
				// 1 because do not want to count the trailing quote and
				// subtract another 1 because already consumed u but have yet to
				// increment i.
				if i > buflen-6 {
					return "", buf[i+1:], fmt.Errorf("cannot decode string: %s", io.ErrShortBuffer)
				}
				v, err := parseUint64FromHexSlice(buf[i+1 : i+5])
				if err != nil {
					return "", buf[i+1:], fmt.Errorf("cannot decode string: %s", err)
				}
				i += 4 // absorb 4 characters: one 'u' and three of the digits

				nbl := len(newBytes)
				newBytes = append(newBytes, 0, 0, 0, 0) // grow to make room for UTF-8 encoded rune

				r := rune(v)
				if utf16.IsSurrogate(r) {
					i++ // absorb final hexidecimal digit from previous value

					// Expect second half of surrogate pair
					if i > buflen-6 || buf[i] != '\\' || buf[i+1] != 'u' {
						return "", buf[i+1:], errors.New("cannot decode string: missing second half of surrogate pair")
					}

					v, err = parseUint64FromHexSlice(buf[i+2 : i+6])
					if err != nil {
						return "", buf[i+1:], fmt.Errorf("cannot decode string: cannot decode second half of surrogate pair: %s", err)
					}
					i += 5 // absorb 5 characters: two for '\u', and 3 of the 4 digits

					// Get code point by combining high and low surrogate bits
					r = utf16.DecodeRune(r, rune(v))
				}

				width := utf8.EncodeRune(newBytes[nbl:], r) // append UTF-8 encoded version of code point
				newBytes = newBytes[:nbl+width]             // trim off excess bytes
				continue
			}
			newBytes = append(newBytes, b)
			continue
		}
		if b == '\\' {
			escaped = true
			continue
		}
		if b == '"' {
			return string(newBytes), buf[i+1:], nil
		}
		newBytes = append(newBytes, b)
	}
	return "", buf, fmt.Errorf("cannot decode string: expected final '\"'; found: %#U", buf[buflen-1])
}

// parseUint64FromHexSlice decodes four characters as hexidecimal digits into a
// uint64 value. It returns an error when any of the four characters are not
// valid hexidecimal digits.
func parseUint64FromHexSlice(buf []byte) (uint64, error) {
	var value uint64
	for _, b := range buf {
		diff := uint64(b - '0')
		if diff < 0 {
			return 0, hex.InvalidByteError(b)
		}
		if diff < 10 {
			// digit 0-9
			value = (value << 4) | diff
			continue
		}
		// letter a-f or A-F
		b10 := b + 10
		diff = uint64(b10 - 'A')
		if diff < 10 {
			return 0, hex.InvalidByteError(b)
		}
		if diff < 16 {
			// letter A-F
			value = (value << 4) | diff
			continue
		}
		// letter a-f
		diff = uint64(b10 - 'a')
		if diff < 10 {
			return 0, hex.InvalidByteError(b)
		}
		if diff < 16 {
			value = (value << 4) | diff
			continue
		}
		return 0, hex.InvalidByteError(b)
	}
	return value, nil
}

// unescapeSpecialJSON attempts to decode one of 8 special bytes. It returns the
// decoded byte and true if the original byte was one of the 8; otherwise it
// returns the original byte and false.
func unescapeSpecialJSON(b byte) (byte, bool) {
	// NOTE: The following 8 special JSON characters must be escaped:
	switch b {
	case '"', '\\', '/':
		return b, true
	case 'b':
		return '\b', true
	case 'f':
		return '\f', true
	case 'n':
		return '\n', true
	case 'r':
		return '\r', true
	case 't':
		return '\t', true
	}
	return b, false
}
