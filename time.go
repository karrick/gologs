package gologs

import (
	"strconv"
	"time"
)

// TimeFormat returns a time formatter that appends the current time to buf as
// a JSON property name and value using the specified string format.
func TimeFormat(format string) func([]byte) []byte {
	return func(buf []byte) []byte {
		return appendString(buf, "time", time.Now().Format(format))
	}
}

// TimeRFC3339 appends the current time to buf as a JSON property name and
// value.
func TimeRFC3339(buf []byte) []byte {
	return appendString(buf, "time", time.Now().Format(time.RFC3339))
}

// TimeUnix appends the current Unix second time to buf as a JSON property
// name and value.
func TimeUnix(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
	return append(buf, ',')
}

// TimeUnixMilli appends the current Unix millisecond time to buf as a JSON
// property name and value.
func TimeUnixMilli(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixMilli(), 10)
	return append(buf, ',')
}

// TimeUnixMicro appends the current Unix microsecond time to buf as a JSON
// property name and value.
func TimeUnixMicro(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixMicro(), 10)
	return append(buf, ',')
}

// TimeUnixNano appends the current Unix nanosecond time to buf as a JSON
// property name and value.
func TimeUnixNano(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixNano(), 10)
	return append(buf, ',')
}
