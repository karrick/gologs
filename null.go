package gologs

// NullLogger never logs anything.
type NullLogger struct{}

// NewNullLogger returns a Logger that never logs. It is a useful default when
// upstream code does not provide an alternative logger.
//
//     type MyServer struct {
//         logger gologs.Logger
//         // other fields...
//     }
//
//     func NewMyServer() &MyServer {
//         return &MyServer{logger: gologs.NewNullLogger()}
//     }
func NewNullLogger() NullLogger { return NullLogger{} }

func (_ NullLogger) SetLevel(_ Level) {}
func (_ NullLogger) SetQuiet()        {}
func (_ NullLogger) SetVerbose()      {}

func (_ NullLogger) Debug(_ string, _ ...interface{})   {}
func (_ NullLogger) Verbose(_ string, _ ...interface{}) {}
func (_ NullLogger) Info(_ string, _ ...interface{})    {}
func (_ NullLogger) Warning(_ string, _ ...interface{}) {}
func (_ NullLogger) Error(_ string, _ ...interface{})   {}
