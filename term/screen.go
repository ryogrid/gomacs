// Package term provides a pure Go terminal backend using ANSI/VT100 escape sequences.
package term

// Screen is the interface for terminal screen operations.
type Screen interface {
	Init() error
	Fini()
	Size() (int, int)
	PollEvent() Event
	PostEvent(Event)
	Clear()
	SetContent(x, y int, ch rune, style Style)
	Show()
	ShowCursor(x, y int)
	Sync()
}

// Event is the interface for terminal events.
type Event interface {
	isEvent()
}

// KeyEvent represents a keyboard input event.
type KeyEvent struct {
	key  KeyCode
	ch   rune
	mod  ModMask
}

func (e *KeyEvent) isEvent() {}

// Key returns the key code for this event.
func (e *KeyEvent) Key() KeyCode { return e.key }

// Rune returns the rune for this event (valid when Key() == KeyRune).
func (e *KeyEvent) Rune() rune { return e.ch }

// Modifiers returns the modifier mask for this event.
func (e *KeyEvent) Modifiers() ModMask { return e.mod }

// NewKeyEvent creates a new KeyEvent.
func NewKeyEvent(key KeyCode, ch rune, mod ModMask) *KeyEvent {
	return &KeyEvent{key: key, ch: ch, mod: mod}
}

// ResizeEvent represents a terminal resize event.
type ResizeEvent struct {
	width  int
	height int
}

func (e *ResizeEvent) isEvent() {}

// Size returns the new terminal dimensions.
func (e *ResizeEvent) Size() (int, int) { return e.width, e.height }

// NewResizeEvent creates a new ResizeEvent.
func NewResizeEvent(w, h int) *ResizeEvent {
	return &ResizeEvent{width: w, height: h}
}

// Style represents text display style.
type Style uint8

const (
	// StyleDefault is the zero-value default style.
	StyleDefault Style = 0
	styleReverse Style = 1
)

// Reverse returns a new Style with reverse video enabled or disabled.
func (s Style) Reverse(on bool) Style {
	if on {
		return s | styleReverse
	}
	return s &^ styleReverse
}

// IsReverse returns whether reverse video is enabled.
func (s Style) IsReverse() bool {
	return s&styleReverse != 0
}

// ModMask represents keyboard modifier keys.
type ModMask int

const (
	ModNone ModMask = 0
	ModAlt  ModMask = 1
)

// KeyCode represents a keyboard key.
type KeyCode int

const (
	KeyRune KeyCode = iota + 256
	KeyNUL
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlI
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlQ
	KeyCtrlR
	KeyCtrlS
	KeyCtrlT
	KeyCtrlU
	KeyCtrlV
	KeyCtrlW
	KeyCtrlX
	KeyCtrlY
	KeyCtrlZ
	KeyCtrlSpace
	KeyCtrlUnderscore
	KeyEnter
	KeyBackspace
	KeyBackspace2
	KeyEsc
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyTab
)
