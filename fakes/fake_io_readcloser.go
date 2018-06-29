package fakes

import (
	"errors"
)

// HappyIOReadCloser is a stub-in for response.Body
type HappyIOReadCloser struct{}

// Close implements io.Closer
func (closer HappyIOReadCloser) Close() error {
	return nil
}

// Read implements io.Reader
func (closer HappyIOReadCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

// SadIOCloser is a stub-in for response.Body
type SadIOCloser struct{}

// Close implements io.Closer
func (closer SadIOCloser) Close() error {
	return errors.New("CLOSE FAIL")
}

// Read implements io.Reader
func (closer SadIOCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

// SadIOReader is a stub-in for response.Body
type SadIOReader struct {
	reads int
}

// Close implements io.Closer
func (closer SadIOReader) Close() error {
	return nil
}

// Read implements io.Reader
func (closer *SadIOReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("READ FAIL")
}
