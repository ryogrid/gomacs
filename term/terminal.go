package term

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
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
	tcgets     = 0x5401 // TCGETS ioctl number on Linux
	tcsets     = 0x5402 // TCSETS ioctl number on Linux
	tiocgwinsz = 0x5413 // TIOCGWINSZ ioctl number on Linux
)

// winsize mirrors the C struct winsize used by TIOCGWINSZ.
type winsize struct {
	Row uint16
	Col uint16
	X   uint16
	Y   uint16
}

// cell represents a single character cell on the screen.
type cell struct {
	ch    rune
	style Style
}

// Terminal is a pure Go terminal backend implementing the Screen interface.
type Terminal struct {
	origTermios *termios
	fd          int
	events      chan Event
	sigwinch    chan os.Signal
	stopSig     chan struct{}
	// Screen buffer
	cells    [][]cell // current buffer [row][col]
	prev     [][]cell // previous buffer for diffing
	width    int
	height   int
	cursorX  int
	cursorY  int
	out      *bufio.Writer
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

	// Set up buffered output.
	t.out = bufio.NewWriter(os.Stdout)

	// Initialize screen buffers.
	t.width, t.height = t.Size()
	t.cells = makeBuffer(t.width, t.height)
	t.prev = makeBuffer(t.width, t.height)
	// Mark all prev cells as different so first Show() draws everything.
	for r := range t.prev {
		for c := range t.prev[r] {
			t.prev[r][c].ch = -1 // sentinel: never matches
		}
	}

	// Set up event channel and SIGWINCH handler.
	t.events = make(chan Event, 64)
	t.sigwinch = make(chan os.Signal, 1)
	t.stopSig = make(chan struct{})
	signal.Notify(t.sigwinch, syscall.SIGWINCH)
	go t.handleSigwinch()

	return nil
}

// Fini restores the terminal to its original state.
func (t *Terminal) Fini() {
	// Stop SIGWINCH handler.
	if t.stopSig != nil {
		close(t.stopSig)
		signal.Stop(t.sigwinch)
	}

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

// handleSigwinch listens for SIGWINCH signals and posts ResizeEvents.
func (t *Terminal) handleSigwinch() {
	for {
		select {
		case <-t.stopSig:
			return
		case <-t.sigwinch:
			w, h := t.Size()
			ev := NewResizeEvent(w, h)
			select {
			case t.events <- ev:
			default:
				// Drop resize event if channel is full.
			}
		}
	}
}

// Size returns the current terminal dimensions (columns, rows) using TIOCGWINSZ.
func (t *Terminal) Size() (int, int) {
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(t.fd),
		uintptr(tiocgwinsz),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 || ws.Col == 0 || ws.Row == 0 {
		return 80, 24 // fallback
	}
	return int(ws.Col), int(ws.Row)
}

// makeBuffer allocates a screen buffer of the given size filled with spaces.
func makeBuffer(width, height int) [][]cell {
	buf := make([][]cell, height)
	for r := range buf {
		buf[r] = make([]cell, width)
		for c := range buf[r] {
			buf[r][c] = cell{ch: ' ', style: StyleDefault}
		}
	}
	return buf
}

// Clear resets all cells to space with default style.
func (t *Terminal) Clear() {
	for r := range t.cells {
		for c := range t.cells[r] {
			t.cells[r][c] = cell{ch: ' ', style: StyleDefault}
		}
	}
}

// SetContent sets a cell in the buffer. Out-of-bounds writes are ignored.
func (t *Terminal) SetContent(x, y int, ch rune, style Style) {
	if y >= 0 && y < t.height && x >= 0 && x < t.width {
		t.cells[y][x] = cell{ch: ch, style: style}
	}
}

// Show diffs current buffer vs previous buffer and writes only changed cells.
func (t *Terminal) Show() {
	// Resize buffers if terminal size changed.
	w, h := t.Size()
	if w != t.width || h != t.height {
		t.resize(w, h)
	}

	for r := 0; r < t.height; r++ {
		for c := 0; c < t.width; c++ {
			cur := t.cells[r][c]
			if cur != t.prev[r][c] {
				// Move cursor to position (ANSI is 1-based).
				t.out.WriteString("\033[")
				t.out.WriteString(strconv.Itoa(r + 1))
				t.out.WriteByte(';')
				t.out.WriteString(strconv.Itoa(c + 1))
				t.out.WriteByte('H')
				// Apply style.
				if cur.style.IsReverse() {
					t.out.WriteString("\033[7m")
				}
				// Write character.
				t.out.WriteRune(cur.ch)
				// Reset style if we applied reverse.
				if cur.style.IsReverse() {
					t.out.WriteString("\033[0m")
				}
				t.prev[r][c] = cur
			}
		}
	}

	// Position hardware cursor and show it.
	t.out.WriteString("\033[")
	t.out.WriteString(strconv.Itoa(t.cursorY + 1))
	t.out.WriteByte(';')
	t.out.WriteString(strconv.Itoa(t.cursorX + 1))
	t.out.WriteByte('H')
	t.out.WriteString("\033[?25h") // show cursor

	t.out.Flush()
}

// ShowCursor positions the hardware cursor.
func (t *Terminal) ShowCursor(x, y int) {
	t.cursorX = x
	t.cursorY = y
}

// Sync forces a full screen redraw by marking all previous cells as dirty.
func (t *Terminal) Sync() {
	for r := range t.prev {
		for c := range t.prev[r] {
			t.prev[r][c].ch = -1 // sentinel
		}
	}
}

// resize adjusts the screen buffers to a new terminal size.
func (t *Terminal) resize(w, h int) {
	t.width = w
	t.height = h
	t.cells = makeBuffer(w, h)
	t.prev = makeBuffer(w, h)
	// Mark all prev dirty so everything redraws.
	for r := range t.prev {
		for c := range t.prev[r] {
			t.prev[r][c].ch = -1
		}
	}
}

// Stub methods for events (to be implemented in US-005).

func (t *Terminal) PollEvent() Event  { return nil }
func (t *Terminal) PostEvent(ev Event) {}
