# PRD: Horizontal Window Splitting (Side-by-Side Windows)

## Introduction

Currently, goomacs only supports vertical stacking of windows (top/bottom via `C-x 2`). This feature adds horizontal splitting (`C-x 3`) so that windows can be arranged side-by-side. The split mode is uniform — all windows in a session share the same split orientation (either all vertical or all horizontal), keeping the implementation simple with the existing flat window list.

## Goals

- Allow users to split windows horizontally (side-by-side) using `C-x 3`
- Provide a clear vertical separator between side-by-side windows
- Maintain the existing `C-x 2` vertical split behavior
- Keep the flat window list architecture (no tree-based layout)

## User Stories

### US-001: Add split orientation state
**Description:** As a developer, I need to track whether the current window layout is vertical (top/bottom) or horizontal (left/right) so the layout engine knows how to arrange windows.

**Acceptance Criteria:**
- [ ] Add a split orientation variable (e.g., `splitMode` with values `vertical` or `horizontal`)
- [ ] Default split mode is `vertical` (preserving current behavior)
- [ ] When only one window exists, the split mode resets to allow either split type next
- [ ] Typecheck/lint passes

### US-002: Implement C-x 3 horizontal split command
**Description:** As a user, I want to press `C-x 3` to split the current window side-by-side so I can view two buffers next to each other.

**Acceptance Criteria:**
- [ ] `C-x 3` creates a new window to the right of the active window, showing the same buffer
- [ ] The new window inherits the same buffer and scroll offset as the active window
- [ ] If the current split mode is `vertical` and there are already multiple windows, `C-x 3` is rejected with a message (e.g., "Cannot mix split orientations")
- [ ] If only one window exists, `C-x 3` sets split mode to `horizontal` and creates the split
- [ ] Screen width is divided evenly among all horizontal windows (minus separator columns)
- [ ] Typecheck/lint passes

### US-003: Update layout calculation for horizontal splits
**Description:** As a developer, I need `recalcWindows` to handle horizontal layout so that side-by-side windows are correctly positioned and sized.

**Acceptance Criteria:**
- [ ] When split mode is `horizontal`, each window gets a `StartCol` and `Width` instead of only `StartRow` and `Height`
- [ ] Add `StartCol` and `Width` fields to the `Window` struct
- [ ] Available screen width is divided evenly among windows, accounting for 1-column vertical separators between adjacent windows
- [ ] Each horizontal window spans the full screen height (minus the shared message line)
- [ ] Remainder columns are distributed to the first windows (same approach as vertical height distribution)
- [ ] Typecheck/lint passes

### US-004: Render side-by-side window content
**Description:** As a user, I want each side-by-side window to display its buffer content within its allocated screen region so I can read both buffers independently.

**Acceptance Criteria:**
- [ ] Buffer text is drawn within each window's column range (`StartCol` to `StartCol + Width`)
- [ ] Lines longer than the window width are truncated at the window boundary (no overflow into adjacent windows)
- [ ] Each window has its own status line at the bottom of its region
- [ ] The active window's status line uses reverse video (`==`); inactive windows use dashes (`--`)
- [ ] Syntax highlighting works correctly within each window's bounds
- [ ] Typecheck/lint passes

### US-005: Draw vertical separator between horizontal windows
**Description:** As a user, I want to see a clear vertical line between side-by-side windows so I can distinguish window boundaries.

**Acceptance Criteria:**
- [ ] A `│` (U+2502 BOX DRAWINGS LIGHT VERTICAL) character is drawn in the separator column between adjacent horizontal windows
- [ ] The separator spans from the top row to the bottom of the window area (excluding the message line)
- [ ] The separator is visually distinct (e.g., default terminal foreground color)
- [ ] Typecheck/lint passes

### US-006: Handle cursor positioning in horizontal windows
**Description:** As a user, I want the cursor to appear at the correct position within whichever side-by-side window is active.

**Acceptance Criteria:**
- [ ] The hardware cursor is placed at the correct (row, col) offset within the active horizontal window
- [ ] Cursor column accounts for the window's `StartCol` offset
- [ ] Cursor is not visible in inactive windows
- [ ] Typecheck/lint passes

### US-007: Adapt C-x o, C-x 0, C-x 1 for horizontal splits
**Description:** As a user, I want existing window management commands to work correctly with side-by-side windows.

**Acceptance Criteria:**
- [ ] `C-x o` cycles through horizontal windows left-to-right, wrapping around
- [ ] `C-x 0` closes the active horizontal window and redistributes width to remaining windows
- [ ] `C-x 1` closes all windows except the active one and resets split mode (allowing either orientation next)
- [ ] When closing brings window count to 1, split mode resets
- [ ] Typecheck/lint passes

### US-008: Handle terminal resize for horizontal splits
**Description:** As a user, I want side-by-side windows to resize proportionally when the terminal is resized.

**Acceptance Criteria:**
- [ ] On terminal resize event, `recalcWindows` is called and redistributes width evenly
- [ ] Windows remain usable after resize (minimum width enforced, e.g., 10 columns)
- [ ] If terminal is too narrow for all windows, excess windows are closed with a warning message
- [ ] Typecheck/lint passes

## Functional Requirements

- FR-1: Add `StartCol` and `Width` fields to the `Window` struct
- FR-2: Add a `splitMode` variable tracking the current orientation (`vertical` or `horizontal`)
- FR-3: `C-x 3` creates a horizontal split; rejected if current mode is `vertical` with multiple windows
- FR-4: `recalcWindows` handles both orientations: vertical distributes height, horizontal distributes width (minus separator columns)
- FR-5: In horizontal mode, draw buffer content within each window's column range, truncating lines at the window boundary
- FR-6: Draw `│` separator characters between adjacent horizontal windows
- FR-7: Position the hardware cursor correctly within the active window's column region
- FR-8: `C-x 0` and `C-x 1` reset `splitMode` when window count drops to 1
- FR-9: Terminal resize redistributes width evenly in horizontal mode
- FR-10: Enforce a minimum window width (e.g., 10 columns); reject splits that would make windows too narrow

## Non-Goals

- No mixed/nested split layouts (no combining vertical and horizontal splits simultaneously)
- No tree-based window layout data structure
- No directional window navigation (e.g., move left/right/up/down between windows)
- No mouse-based window resizing or dragging separators
- No uneven/manual window sizing

## Technical Considerations

- The `Window` struct gains `StartCol` and `Width` fields; in vertical mode these default to `0` and `screenWidth` respectively
- `recalcWindows` needs a mode parameter or access to `splitMode` to decide layout direction
- `drawWindowContent` must be updated to respect column boundaries — each `SetContent` call must be offset by `StartCol` and lines must be clipped at `StartCol + Width`
- The `drawWindowStatusLine` function must also be constrained to the window's column range in horizontal mode
- The message line remains shared across the full screen width at the bottom row

## Success Metrics

- Users can split windows side-by-side with `C-x 3` and switch between them with `C-x o`
- All existing vertical split functionality continues to work unchanged
- No rendering artifacts or overflow between adjacent horizontal windows

## Open Questions

- Should `C-x 2` be allowed when in horizontal mode with multiple windows? (Current plan: reject with message, same as `C-x 3` in vertical mode)
- Should there be a command to toggle/switch split orientation for all windows at once?
