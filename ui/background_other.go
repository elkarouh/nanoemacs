//go:build !unix

package ui

// DetectBackground is a no-op where there is no /dev/tty to ask.
func DetectBackground() {}
