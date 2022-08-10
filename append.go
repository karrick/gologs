package gologs

import (
	"fmt"
	"strconv"
)

func appendBool(buf []byte, name string, value bool) []byte {
	buf = appendEncodedJSONFromString(buf, name)
	buf = append(buf, ':')
	if value {
		return append(buf, []byte("true,")...)
	}
	return append(buf, []byte("false,")...)
}

func appendFloat(buf []byte, name string, value float64) []byte {
	buf = appendEncodedJSONFromString(buf, name)
	buf = append(buf, ':')
	buf = appendEncodedJSONFromFloat(buf, value)
	return append(buf, ',')
}

func appendFormat(buf []byte, name, f string, args ...interface{}) []byte {
	return appendString(buf, name, fmt.Sprintf(f, args...))
}

func appendInt(buf []byte, name string, value int64) []byte {
	buf = appendEncodedJSONFromString(buf, name)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, value, 10)
	return append(buf, ',')
}

func appendString(buf []byte, name, value string) []byte {
	buf = appendEncodedJSONFromString(buf, name)
	buf = append(buf, ':')
	buf = appendEncodedJSONFromString(buf, value)
	return append(buf, ',')
}

func appendUint(buf []byte, name string, value uint64) []byte {
	buf = appendEncodedJSONFromString(buf, name)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, value, 10)
	return append(buf, ',')
}
