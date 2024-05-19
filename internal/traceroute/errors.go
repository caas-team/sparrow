package traceroute

import "fmt"

// ErrClosingConn represents an error that occurred while closing a connection
type ErrClosingConn struct {
	Err error
}

func (e ErrClosingConn) Error() string {
	return fmt.Sprintf("error closing connection: %v", e.Err)
}

// Unwrap returns the wrapped error
func (e ErrClosingConn) Unwrap() error {
	return e.Err
}

// Is checks if the target error is an ErrClosingConn
func (e ErrClosingConn) Is(target error) bool {
	_, ok := target.(*ErrClosingConn)
	return ok
}
