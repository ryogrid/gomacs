package term

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/mattn/go-runewidth"
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
	posted      chan Event // re-queued events from PostEvent
	sigwinch    chan os.Signal
	stopSig     chan struct{}
	in          io.Reader // input source (os.Stdin by default)
	// Screen buffer
	cells   [][]cell // current buffer [row][col]
	prev    [][]cell // previous buffer for diffing
	width   int
	height  int
	cursorX int
	cursorY int
	out     *bufio.Writer
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

	// Set up event channels and SIGWINCH handler.
	t.events = make(chan Event, 64)
	t.posted = make(chan Event, 64)
	t.sigwinch = make(chan os.Signal, 1)
	t.stopSig = make(chan struct{})
	t.in = os.Stdin
	signal.Notify(t.sigwinch, syscall.SIGWINCH)
	go t.handleSigwinch()
	go t.readInput()

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

// writeStyledCell writes ANSI escape sequences for a cell's style, the character,
// and a reset sequence if any style attributes were applied.
func (t *Terminal) writeStyledCell(ch rune, style Style) {
	styled := style.IsBold() || style.Fg() != ColorDefault || style.Bg() != ColorDefault || style.IsReverse()
	if styled {
		t.out.WriteString("\033[")
		first := true
		if style.IsBold() {
			t.out.WriteByte('1')
			first = false
		}
		if style.Fg() != ColorDefault {
			if !first {
				t.out.WriteByte(';')
			}
			t.out.WriteString("38;5;")
			t.out.WriteString(strconv.Itoa(int(style.Fg())))
			first = false
		}
		if style.Bg() != ColorDefault {
			if !first {
				t.out.WriteByte(';')
			}
			t.out.WriteString("48;5;")
			t.out.WriteString(strconv.Itoa(int(style.Bg())))
			first = false
		}
		if style.IsReverse() {
			if !first {
				t.out.WriteByte(';')
			}
			t.out.WriteByte('7')
		}
		t.out.WriteByte('m')
	}
	t.out.WriteRune(ch)
	if styled {
		t.out.WriteString("\033[0m")
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
				t.writeStyledCell(cur.ch, cur.style)
				t.prev[r][c] = cur
				// Wide character: skip the next column (covered by this char).
				if runewidth.RuneWidth(cur.ch) == 2 && c+1 < t.width {
					t.prev[r][c+1] = t.cells[r][c+1]
					c++
				}
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

// PollEvent blocks until an event is available and returns it.
// Re-queued events (via PostEvent) are returned before stdin events.
func (t *Terminal) PollEvent() Event {
	select {
	case ev := <-t.posted:
		return ev
	default:
	}
	select {
	case ev := <-t.posted:
		return ev
	case ev := <-t.events:
		return ev
	}
}

// PostEvent re-queues an event to be returned by the next PollEvent() call.
func (t *Terminal) PostEvent(ev Event) {
	select {
	case t.posted <- ev:
	default:
	}
}

// escTimeout is how long to wait after receiving an Esc byte to distinguish
// a bare Escape press from an Alt+key or ANSI escape sequence.
const escTimeout = 50 * time.Millisecond

// readInput reads from the input source, parses key events, and sends them
// to the events channel. It runs in a goroutine started by Init().
func (t *Terminal) readInput() {
	buf := make([]byte, 256)
	for {
		n, err := t.in.Read(buf)
		if err != nil || n == 0 {
			return
		}
		t.parseInput(buf[:n])
	}
}

// parseInput processes a chunk of bytes into key events.
func (t *Terminal) parseInput(data []byte) {
	i := 0
	for i < len(data) {
		b := data[i]

		if b == 0x1b { // Escape
			// Check if there are more bytes in this chunk that form a sequence.
			if i+1 < len(data) {
				i = t.parseEscSequence(data, i)
				continue
			}
			// No more bytes in buffer; wait briefly for more.
			extra := make([]byte, 16)
			done := make(chan int, 1)
			go func() {
				n, _ := t.in.Read(extra)
				done <- n
			}()
			select {
			case n := <-done:
				if n > 0 {
					// Combine with extra bytes and parse as escape sequence.
					combined := append([]byte{0x1b}, extra[:n]...)
					t.parseEscSequence(combined, 0)
				} else {
					t.sendEvent(NewKeyEvent(KeyEsc, 0, ModNone))
				}
			case <-time.After(escTimeout):
				// Timeout: bare Escape key.
				t.sendEvent(NewKeyEvent(KeyEsc, 0, ModNone))
				// The goroutine will eventually read and we need to handle those bytes.
				go func() {
					n := <-done
					if n > 0 {
						t.parseInput(extra[:n])
					}
				}()
			}
			i++
			continue
		}

		if b == 0x00 { // C-SPC / NUL
			t.sendEvent(NewKeyEvent(KeyCtrlSpace, 0, ModNone))
			i++
			continue
		}

		if b == 0x09 { // Tab (C-i)
			t.sendEvent(NewKeyEvent(KeyTab, 0, ModNone))
			i++
			continue
		}

		if b == 0x0d { // Enter (C-m)
			t.sendEvent(NewKeyEvent(KeyEnter, 0, ModNone))
			i++
			continue
		}

		if b == 0x1f { // C-_
			t.sendEvent(NewKeyEvent(KeyCtrlUnderscore, 0, ModNone))
			i++
			continue
		}

		if b == 0x7f { // Backspace
			t.sendEvent(NewKeyEvent(KeyBackspace2, 0, ModNone))
			i++
			continue
		}

		if b >= 0x01 && b <= 0x1a { // C-a through C-z
			// Map 0x01 -> KeyCtrlA, 0x02 -> KeyCtrlB, etc.
			key := KeyCtrlA + KeyCode(b-0x01)
			t.sendEvent(NewKeyEvent(key, 0, ModNone))
			i++
			continue
		}

		// Multi-byte UTF-8 or ASCII printable character.
		if b >= 0x20 && b < 0x7f {
			t.sendEvent(NewKeyEvent(KeyRune, rune(b), ModNone))
			i++
			continue
		}

		// Multi-byte UTF-8 sequence.
		if b >= 0x80 {
			r, size := utf8.DecodeRune(data[i:])
			if r == utf8.RuneError && size <= 1 {
				i++
				continue
			}
			t.sendEvent(NewKeyEvent(KeyRune, r, ModNone))
			i += size
			continue
		}

		// Unknown byte, skip.
		i++
	}
}

// parseEscSequence parses an escape sequence starting at data[i] (which is 0x1b).
// Returns the index past the consumed bytes.
func (t *Terminal) parseEscSequence(data []byte, i int) int {
	if i+1 >= len(data) {
		t.sendEvent(NewKeyEvent(KeyEsc, 0, ModNone))
		return i + 1
	}

	next := data[i+1]

	// CSI sequence: ESC [
	if next == '[' {
		if i+2 < len(data) {
			switch data[i+2] {
			case 'A':
				t.sendEvent(NewKeyEvent(KeyUp, 0, ModNone))
				return i + 3
			case 'B':
				t.sendEvent(NewKeyEvent(KeyDown, 0, ModNone))
				return i + 3
			case 'C':
				t.sendEvent(NewKeyEvent(KeyRight, 0, ModNone))
				return i + 3
			case 'D':
				t.sendEvent(NewKeyEvent(KeyLeft, 0, ModNone))
				return i + 3
			}
		}
		// Unknown CSI sequence, consume ESC [.
		t.sendEvent(NewKeyEvent(KeyEsc, 0, ModNone))
		return i + 1
	}

	// Alt+key: ESC followed by printable character or control character.
	if next >= 0x20 && next < 0x7f {
		t.sendEvent(NewKeyEvent(KeyRune, rune(next), ModAlt))
		return i + 2
	}

	// Alt + control character
	if next >= 0x01 && next <= 0x1a {
		key := KeyCtrlA + KeyCode(next-0x01)
		t.sendEvent(NewKeyEvent(key, 0, ModAlt))
		return i + 2
	}

	// Unknown escape sequence, send bare Esc.
	t.sendEvent(NewKeyEvent(KeyEsc, 0, ModNone))
	return i + 1
}

// sendEvent sends an event to the events channel without blocking.
func (t *Terminal) sendEvent(ev Event) {
	select {
	case t.events <- ev:
	default:
	}
}
