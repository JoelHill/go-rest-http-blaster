package fakes

// TimeoutError is a stub in for net.Error with a timeout
type TimeoutError struct{}

// Timeout implements net.timeout
func (t TimeoutError) Timeout() bool {
	return true
}

// Temporary implements net.temporary
func (t TimeoutError) Temporary() bool {
	return false
}

// Error implements error
func (t TimeoutError) Error() string {
	return "timeout"
}
