package term

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// termios mirrors the C struct termios for terminal I/O settings.
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Line   uint8
	Cc     [32]uint8
	Ispeed uint32
	Ospeed uint32
}

const (
	tcgets = 0x5401 // TCGETS ioctl number on Linux
	tcsets = 0x5402 // TCSETS ioctl number on Linux
)

// Terminal is a pure Go terminal backend implementing the Screen interface.
type Terminal struct {
	origTermios *termios
	fd          int
}

// NewTerminal creates a new Terminal instance.
func NewTerminal() *Terminal {
	return &Terminal{
		fd: int(os.Stdin.Fd()),
	}
}

// Init puts the terminal into raw mode and enters the alternate screen buffer.
func (t *Terminal) Init() error {
	// Save original termios state.
	orig, err := t.getTermios()
	if err != nil {
		return fmt.Errorf("term: failed to get termios: %w", err)
	}
	t.origTermios = orig

	// Set raw mode.
	raw := *orig
	// Input flags: disable BRKINT, ICRNL, INPCK, ISTRIP, IXON
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	// Output flags: disable OPOST
	raw.Oflag &^= syscall.OPOST
	// Control flags: set CS8
	raw.Cflag |= syscall.CS8
	// Local flags: disable ECHO, ICANON, IEXTEN, ISIG
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	// Control characters: read returns after 1 byte, no timeout
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := t.setTermios(&raw); err != nil {
		return fmt.Errorf("term: failed to set raw mode: %w", err)
	}

	// Enter alternate screen buffer and hide cursor.
	os.Stdout.WriteString("\033[?1049h") // enter alternate screen
	os.Stdout.WriteString("\033[?25l")   // hide cursor

	return nil
}

// Fini restores the terminal to its original state.
func (t *Terminal) Fini() {
	// Exit alternate screen buffer and show cursor.
	os.Stdout.WriteString("\033[?25h")   // show cursor
	os.Stdout.WriteString("\033[?1049l") // exit alternate screen

	// Restore original termios.
	if t.origTermios != nil {
		t.setTermios(t.origTermios)
	}
}

// getTermios reads the current terminal settings.
func (t *Terminal) getTermios() (*termios, error) {
	var tio termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		uintptr(tcgets),
		uintptr(unsafe.Pointer(&tio)),
	)
	if errno != 0 {
		return nil, errno
	}
	return &tio, nil
}

// setTermios applies terminal settings.
func (t *Terminal) setTermios(tio *termios) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		uintptr(tcsets),
		uintptr(unsafe.Pointer(tio)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Stub methods to satisfy the Screen interface.

func (t *Terminal) Size() (int, int)               { return 80, 24 }
func (t *Terminal) PollEvent() Event               { return nil }
func (t *Terminal) PostEvent(ev Event)              {}
func (t *Terminal) Clear()                          {}
func (t *Terminal) SetContent(x, y int, ch rune, style Style) {}
func (t *Terminal) Show()                           {}
func (t *Terminal) ShowCursor(x, y int)             {}
func (t *Terminal) Sync()                           {}
