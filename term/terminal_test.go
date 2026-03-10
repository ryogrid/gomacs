package term

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

// showForTest is like Show() but skips the Size() syscall,
// using the already-set width/height values.
func (t *Terminal) showForTest() {
	for r := 0; r < t.height; r++ {
		for c := 0; c < t.width; c++ {
			cur := t.cells[r][c]
			if cur != t.prev[r][c] {
				t.out.WriteString("\033[")
				t.out.WriteString(strconv.Itoa(r + 1))
				t.out.WriteByte(';')
				t.out.WriteString(strconv.Itoa(c + 1))
				t.out.WriteByte('H')
				if cur.style.IsReverse() {
					t.out.WriteString("\033[7m")
				}
				t.out.WriteRune(cur.ch)
				if cur.style.IsReverse() {
					t.out.WriteString("\033[0m")
				}
				t.prev[r][c] = cur
			}
		}
	}
	t.out.WriteString("\033[")
	t.out.WriteString(strconv.Itoa(t.cursorY + 1))
	t.out.WriteByte(';')
	t.out.WriteString(strconv.Itoa(t.cursorX + 1))
	t.out.WriteByte('H')
	t.out.WriteString("\033[?25h")
	t.out.Flush()
}

// newTestTerminal creates a Terminal with channels and buffers initialized
// for testing (no real terminal needed).
func newTestTerminal(width, height int) *Terminal {
	t := &Terminal{
		events:  make(chan Event, 64),
		posted:  make(chan Event, 64),
		width:   width,
		height:  height,
		cells:   makeBuffer(width, height),
		prev:    makeBuffer(width, height),
		cursorX: 0,
		cursorY: 0,
	}
	return t
}

// drainEvents reads all available events from the events channel.
func drainEvents(t *Terminal) []Event {
	var events []Event
	for {
		select {
		case ev := <-t.events:
			events = append(events, ev)
		default:
			return events
		}
	}
}

// --- Keyboard Input Parsing: Control Characters ---

func TestParseInput_ControlChars(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		key  KeyCode
	}{
		{"C-a", 0x01, KeyCtrlA},
		{"C-b", 0x02, KeyCtrlB},
		{"C-c", 0x03, KeyCtrlC},
		{"C-d", 0x04, KeyCtrlD},
		{"C-e", 0x05, KeyCtrlE},
		{"C-f", 0x06, KeyCtrlF},
		{"C-g", 0x07, KeyCtrlG},
		{"C-h", 0x08, KeyCtrlH},
		{"C-k", 0x0b, KeyCtrlK},
		{"C-l", 0x0c, KeyCtrlL},
		{"C-n", 0x0e, KeyCtrlN},
		{"C-o", 0x0f, KeyCtrlO},
		{"C-p", 0x10, KeyCtrlP},
		{"C-q", 0x11, KeyCtrlQ},
		{"C-r", 0x12, KeyCtrlR},
		{"C-s", 0x13, KeyCtrlS},
		{"C-t", 0x14, KeyCtrlT},
		{"C-u", 0x15, KeyCtrlU},
		{"C-v", 0x16, KeyCtrlV},
		{"C-w", 0x17, KeyCtrlW},
		{"C-x", 0x18, KeyCtrlX},
		{"C-y", 0x19, KeyCtrlY},
		{"C-z", 0x1a, KeyCtrlZ},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal(80, 24)
			term.parseInput([]byte{tt.b})
			events := drainEvents(term)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			ke, ok := events[0].(*KeyEvent)
			if !ok {
				t.Fatalf("expected *KeyEvent, got %T", events[0])
			}
			if ke.Key() != tt.key {
				t.Errorf("expected key %d, got %d", tt.key, ke.Key())
			}
			if ke.Modifiers() != ModNone {
				t.Errorf("expected ModNone, got %d", ke.Modifiers())
			}
		})
	}
}

func TestParseInput_SpecialControlChars(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		key  KeyCode
	}{
		{"NUL/C-SPC", 0x00, KeyCtrlSpace},
		{"Tab", 0x09, KeyTab},
		{"Enter", 0x0d, KeyEnter},
		{"C-_", 0x1f, KeyCtrlUnderscore},
		{"Backspace", 0x7f, KeyBackspace2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal(80, 24)
			term.parseInput([]byte{tt.b})
			events := drainEvents(term)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			ke := events[0].(*KeyEvent)
			if ke.Key() != tt.key {
				t.Errorf("expected key %d, got %d", tt.key, ke.Key())
			}
		})
	}
}

// --- Keyboard Input Parsing: ANSI Escape Sequences ---

func TestParseInput_ArrowKeys(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
		key  KeyCode
	}{
		{"Up", []byte{0x1b, '[', 'A'}, KeyUp},
		{"Down", []byte{0x1b, '[', 'B'}, KeyDown},
		{"Right", []byte{0x1b, '[', 'C'}, KeyRight},
		{"Left", []byte{0x1b, '[', 'D'}, KeyLeft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal(80, 24)
			term.parseInput(tt.seq)
			events := drainEvents(term)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			ke := events[0].(*KeyEvent)
			if ke.Key() != tt.key {
				t.Errorf("expected key %d, got %d", tt.key, ke.Key())
			}
			if ke.Modifiers() != ModNone {
				t.Errorf("expected ModNone, got %d", ke.Modifiers())
			}
		})
	}
}

// --- Keyboard Input Parsing: UTF-8 ---

func TestParseInput_UTF8(t *testing.T) {
	tests := []struct {
		name string
		s    string
		r    rune
	}{
		{"ASCII", "A", 'A'},
		{"2-byte", "é", 'é'},
		{"3-byte", "日", '日'},
		{"4-byte", "🎉", '🎉'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal(80, 24)
			term.parseInput([]byte(tt.s))
			events := drainEvents(term)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			ke := events[0].(*KeyEvent)
			if ke.Key() != KeyRune {
				t.Errorf("expected KeyRune, got %d", ke.Key())
			}
			if ke.Rune() != tt.r {
				t.Errorf("expected rune %q, got %q", tt.r, ke.Rune())
			}
		})
	}
}

// --- Keyboard Input Parsing: Alt+key ---

func TestParseInput_AltKey(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
		r    rune
	}{
		{"Alt-x", []byte{0x1b, 'x'}, 'x'},
		{"Alt-f", []byte{0x1b, 'f'}, 'f'},
		{"Alt-b", []byte{0x1b, 'b'}, 'b'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal(80, 24)
			term.parseInput(tt.seq)
			events := drainEvents(term)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			ke := events[0].(*KeyEvent)
			if ke.Key() != KeyRune {
				t.Errorf("expected KeyRune, got %d", ke.Key())
			}
			if ke.Rune() != tt.r {
				t.Errorf("expected rune %q, got %q", tt.r, ke.Rune())
			}
			if ke.Modifiers() != ModAlt {
				t.Errorf("expected ModAlt, got %d", ke.Modifiers())
			}
		})
	}
}

func TestParseInput_AltControlChar(t *testing.T) {
	// Alt + C-a = Esc followed by 0x01
	term := newTestTerminal(80, 24)
	term.parseInput([]byte{0x1b, 0x01})
	events := drainEvents(term)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ke := events[0].(*KeyEvent)
	if ke.Key() != KeyCtrlA {
		t.Errorf("expected KeyCtrlA, got %d", ke.Key())
	}
	if ke.Modifiers() != ModAlt {
		t.Errorf("expected ModAlt, got %d", ke.Modifiers())
	}
}

// --- Screen Buffer: SetContent and Clear ---

func TestSetContent(t *testing.T) {
	term := newTestTerminal(10, 5)

	term.SetContent(3, 2, 'X', StyleDefault)
	if term.cells[2][3].ch != 'X' {
		t.Errorf("expected 'X' at (3,2), got %q", term.cells[2][3].ch)
	}
	if term.cells[2][3].style != StyleDefault {
		t.Errorf("expected StyleDefault, got %v", term.cells[2][3].style)
	}

	// Reverse style
	rev := StyleDefault.Reverse(true)
	term.SetContent(0, 0, 'R', rev)
	if term.cells[0][0].ch != 'R' || term.cells[0][0].style != rev {
		t.Errorf("expected 'R' with reverse style at (0,0)")
	}
}

func TestSetContent_OutOfBounds(t *testing.T) {
	term := newTestTerminal(10, 5)
	// Should not panic
	term.SetContent(-1, 0, 'X', StyleDefault)
	term.SetContent(0, -1, 'X', StyleDefault)
	term.SetContent(10, 0, 'X', StyleDefault)
	term.SetContent(0, 5, 'X', StyleDefault)
}

func TestClear(t *testing.T) {
	term := newTestTerminal(10, 5)

	// Set some content
	term.SetContent(0, 0, 'A', StyleDefault.Reverse(true))
	term.SetContent(5, 3, 'B', StyleDefault)

	// Clear should reset everything
	term.Clear()

	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			if term.cells[r][c].ch != ' ' || term.cells[r][c].style != StyleDefault {
				t.Errorf("cell (%d,%d) not cleared: ch=%q style=%v", c, r, term.cells[r][c].ch, term.cells[r][c].style)
			}
		}
	}
}

// --- Screen Rendering: Diff ---

func TestShow_OnlyOutputsChangedCells(t *testing.T) {
	term := newTestTerminal(5, 3)
	var buf bytes.Buffer
	term.out = bufio.NewWriter(&buf)

	// Make prev match cells (all spaces) so nothing is dirty
	for r := range term.prev {
		for c := range term.prev[r] {
			term.prev[r][c] = cell{ch: ' ', style: StyleDefault}
		}
	}

	// Change one cell
	term.SetContent(2, 1, 'X', StyleDefault)

	// Override Size() behavior by keeping width/height consistent
	// We call showForTest which doesn't call Size()
	term.showForTest()

	output := buf.String()
	// Should contain positioning for row 2, col 3 (1-based) and the char 'X'
	if !strings.Contains(output, "\033[2;3H") {
		t.Errorf("expected cursor position \\033[2;3H in output, got %q", output)
	}
	if !strings.Contains(output, "X") {
		t.Errorf("expected 'X' in output, got %q", output)
	}

	// Should NOT contain positioning for unchanged cells like (1,1)
	// Count the number of positioning sequences - should be minimal
	posCount := strings.Count(output, "\033[")
	// We expect: 1 cell positioning + cursor positioning + cursor show = at least 2 \033[ sequences
	// but definitely not 15 (5*3 cells)
	if posCount > 5 {
		t.Errorf("too many escape sequences (%d), diff not working properly", posCount)
	}
}

func TestShow_ReverseStyle(t *testing.T) {
	term := newTestTerminal(5, 3)
	var buf bytes.Buffer
	term.out = bufio.NewWriter(&buf)
	// Mark prev dirty
	for r := range term.prev {
		for c := range term.prev[r] {
			term.prev[r][c].ch = -1
		}
	}

	term.SetContent(0, 0, 'R', StyleDefault.Reverse(true))
	term.showForTest()

	output := buf.String()
	if !strings.Contains(output, "\033[7m") {
		t.Errorf("expected reverse video \\033[7m in output")
	}
	if !strings.Contains(output, "\033[0m") {
		t.Errorf("expected reset \\033[0m in output")
	}
}

// --- Style ---

func TestStyle_Reverse(t *testing.T) {
	s := StyleDefault
	if s.IsReverse() {
		t.Error("StyleDefault should not be reverse")
	}

	rev := s.Reverse(true)
	if !rev.IsReverse() {
		t.Error("Reverse(true) should be reverse")
	}

	norev := rev.Reverse(false)
	if norev.IsReverse() {
		t.Error("Reverse(false) should not be reverse")
	}
	if norev != StyleDefault {
		t.Error("Reverse(false) should equal StyleDefault")
	}
}

func TestStyle_Foreground(t *testing.T) {
	s := StyleDefault.Foreground(196)
	if s.Fg() != 196 {
		t.Errorf("expected fg 196, got %d", s.Fg())
	}
	if s.Bg() != ColorDefault {
		t.Errorf("expected bg ColorDefault, got %d", s.Bg())
	}
}

func TestStyle_Background(t *testing.T) {
	s := StyleDefault.Background(21)
	if s.Bg() != 21 {
		t.Errorf("expected bg 21, got %d", s.Bg())
	}
	if s.Fg() != ColorDefault {
		t.Errorf("expected fg ColorDefault, got %d", s.Fg())
	}
}

func TestStyle_Bold(t *testing.T) {
	s := StyleDefault
	if s.IsBold() {
		t.Error("StyleDefault should not be bold")
	}
	b := s.Bold(true)
	if !b.IsBold() {
		t.Error("Bold(true) should be bold")
	}
	nb := b.Bold(false)
	if nb.IsBold() {
		t.Error("Bold(false) should not be bold")
	}
}

func TestStyle_Combined(t *testing.T) {
	s := StyleDefault.Foreground(82).Background(236).Bold(true).Reverse(true)
	if s.Fg() != 82 {
		t.Errorf("expected fg 82, got %d", s.Fg())
	}
	if s.Bg() != 236 {
		t.Errorf("expected bg 236, got %d", s.Bg())
	}
	if !s.IsBold() {
		t.Error("expected bold")
	}
	if !s.IsReverse() {
		t.Error("expected reverse")
	}
}

func TestStyle_DefaultValues(t *testing.T) {
	s := StyleDefault
	if s.Fg() != ColorDefault {
		t.Errorf("expected fg ColorDefault, got %d", s.Fg())
	}
	if s.Bg() != ColorDefault {
		t.Errorf("expected bg ColorDefault, got %d", s.Bg())
	}
	if s.IsBold() {
		t.Error("StyleDefault should not be bold")
	}
	if s.IsReverse() {
		t.Error("StyleDefault should not be reverse")
	}
}

// --- PostEvent ---

func TestPostEvent_PriorityOverStdin(t *testing.T) {
	term := newTestTerminal(80, 24)

	// Put an event in the stdin events channel
	stdinEvent := NewKeyEvent(KeyCtrlA, 0, ModNone)
	term.events <- stdinEvent

	// Post an event (should have priority)
	postedEvent := NewKeyEvent(KeyCtrlB, 0, ModNone)
	term.PostEvent(postedEvent)

	// PollEvent should return the posted event first
	ev := term.PollEvent()
	ke, ok := ev.(*KeyEvent)
	if !ok {
		t.Fatalf("expected *KeyEvent, got %T", ev)
	}
	if ke.Key() != KeyCtrlB {
		t.Errorf("expected posted event (KeyCtrlB) first, got key %d", ke.Key())
	}

	// Next should be the stdin event
	done := make(chan Event, 1)
	go func() {
		done <- term.PollEvent()
	}()
	select {
	case ev := <-done:
		ke := ev.(*KeyEvent)
		if ke.Key() != KeyCtrlA {
			t.Errorf("expected stdin event (KeyCtrlA) second, got key %d", ke.Key())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second event")
	}
}

// --- Sync ---

func TestSync_MarksAllDirty(t *testing.T) {
	term := newTestTerminal(5, 3)

	// Make prev match cells
	for r := range term.prev {
		for c := range term.prev[r] {
			term.prev[r][c] = term.cells[r][c]
		}
	}

	term.Sync()

	// All prev cells should now be dirty (sentinel value)
	for r := range term.prev {
		for c := range term.prev[r] {
			if term.prev[r][c].ch != -1 {
				t.Errorf("cell (%d,%d) not marked dirty after Sync", c, r)
			}
		}
	}
}

// --- Integration: readInput with injected reader ---

func TestReadInput_IntegrationMultipleKeys(t *testing.T) {
	term := newTestTerminal(80, 24)
	// Input: 'H', 'i', Enter
	term.in = bytes.NewReader([]byte{'H', 'i', 0x0d})
	term.readInput()

	events := drainEvents(term)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// 'H'
	ke := events[0].(*KeyEvent)
	if ke.Key() != KeyRune || ke.Rune() != 'H' {
		t.Errorf("event 0: expected 'H', got key=%d rune=%q", ke.Key(), ke.Rune())
	}

	// 'i'
	ke = events[1].(*KeyEvent)
	if ke.Key() != KeyRune || ke.Rune() != 'i' {
		t.Errorf("event 1: expected 'i', got key=%d rune=%q", ke.Key(), ke.Rune())
	}

	// Enter
	ke = events[2].(*KeyEvent)
	if ke.Key() != KeyEnter {
		t.Errorf("event 2: expected KeyEnter, got key=%d", ke.Key())
	}
}

func TestReadInput_EscapeSequences(t *testing.T) {
	term := newTestTerminal(80, 24)
	// Arrow up, then 'a'
	term.in = bytes.NewReader([]byte{0x1b, '[', 'A', 'a'})
	term.readInput()

	events := drainEvents(term)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	ke := events[0].(*KeyEvent)
	if ke.Key() != KeyUp {
		t.Errorf("event 0: expected KeyUp, got %d", ke.Key())
	}

	ke = events[1].(*KeyEvent)
	if ke.Key() != KeyRune || ke.Rune() != 'a' {
		t.Errorf("event 1: expected 'a', got key=%d rune=%q", ke.Key(), ke.Rune())
	}
}
