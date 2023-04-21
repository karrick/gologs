package gologs

// Copied from github.com/karrick/goejs

import (
	"math"
	"strconv"
)

// appendEncodedJSONFromFloat appends the JSON encoded form of the value to
// the provided byte slice. Because some legal IEEE 754 floating point values
// have no JSON equivalents, this library encodes several floating point
// numbers into the corresponding encoded form, as used by several other JSON
// encoding libraries, as shown in the table below.
//
// JSON serialization:
//
//	 NaN: null
//	-Inf: -1e999
//	+Inf: 1e999
func appendEncodedJSONFromFloat(buf []byte, f64 float64) []byte {
	if math.IsNaN(f64) {
		return append(buf, "null"...)
	} else if math.IsInf(f64, 1) {
		return append(buf, "1e999"...)
	} else if math.IsInf(f64, -1) {
		return append(buf, "-1e999"...)
	}
	// NOTE: To support some dynamic languages which will decode a JSON number
	// without a fractional component as a runtime integer, we encode these
	// numbers using exponential notation.
	if f64 == math.Floor(f64) {
		return strconv.AppendFloat(buf, f64, 'e', -1, 64)
	}
	// Otherwise, use the most compact format possible.
	return strconv.AppendFloat(buf, f64, 'g', -1, 64)
}
