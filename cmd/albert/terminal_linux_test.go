package main

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

// openPTY opens a new pseudo-terminal and returns its terminal side.
func openPTY(t *testing.T) *os.File {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("can't open a pseudo-terminal here: %v", err)
	}
	t.Cleanup(func() { master.Close() })
	var unlock int32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
		t.Skipf("can't unlock the pseudo-terminal: %v", errno)
	}
	var n uint32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); errno != 0 {
		t.Skipf("can't name the pseudo-terminal: %v", errno)
	}
	tty, err := os.OpenFile("/dev/pts/"+itoa(n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("can't open the pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { tty.Close() })
	return tty
}

func itoa(n uint32) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func TestIsTerminalAcceptsAPseudoTerminal(t *testing.T) {
	if !isTerminal(openPTY(t)) {
		t.Error("a pseudo-terminal was not treated as a terminal")
	}
}

func TestIsTerminalRejectsDevNull(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if isTerminal(null) {
		t.Error("/dev/null was treated as a terminal")
	}
}
