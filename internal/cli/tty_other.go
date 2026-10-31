//go:build !linux

package cli

// ttyInteractive reports whether input is interactive. On non-linux
// platforms the MVP conservatively assumes no tty (editor is skipped).
func ttyInteractive() bool {
	return false
}
