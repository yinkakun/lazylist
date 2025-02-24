package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type AppMode int
type InputAction int

const (
	ModeInput AppMode = iota + 1
	ModeNormal
)

const (
	ActionEdit InputAction = iota + 1
	ActionCreate
)

type TodoItem struct {
	Title     string
	Completed bool
}

type InputContext struct {
	Cursor     int
	Content    string
	InitialVal string
	Action     InputAction
}

type ValidationError struct {
	Operation string
	Err       error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("todo operation: %s: %v", e.Operation, e.Err)
}

type TodoList struct {
	selectedIndex int
	currentMode   AppMode
	lastErr       error
	items         []TodoItem
	input         InputContext
}

func NewTodoList(initialItems []string) *TodoList {
	listItems := make([]TodoItem, len(initialItems))

	for i, item := range initialItems {
		listItems[i] = TodoItem{
			Title: item,
		}
	}

	return &TodoList{
		items:       listItems,
		currentMode: ModeNormal,
	}
}

// Helper functions

func validateItemTitle(title string) error {
	if len(strings.TrimSpace(title)) == 0 {
		return &ValidationError{Operation: "validate", Err: errors.New("item title cannot be empty")}
	}
	return nil
}

func (t *TodoList) isValidIndex(index int) bool {
	return index >= 0 && index < len(t.items)
}

// normal mode

func (t TodoList) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return t, tea.Quit

	case "up", "k":
		t.moveCursor(CursorUp)

	case "down", "j":
		t.moveCursor(CursorDown)

	case "a":
		t.ToggleAllItems()

	case "enter", " ":
		t.ToggleItem(t.selectedIndex)

	case "n":
		t.enterInputMode(ActionCreate, "")

	case "e":
		if len(t.items) > 0 {
			t.enterInputMode(ActionEdit, t.items[t.selectedIndex].Title)
		}

	case "d":
		if len(t.items) > 0 {
			t.DeleteItem(t.selectedIndex)
		}
	}

	return t, nil
}

func (t *TodoList) adjustCursorAfterDelete() {
	if len(t.items) == 0 {
		t.selectedIndex = 0
	}
	if t.selectedIndex >= len(t.items) {
		t.selectedIndex = len(t.items) - 1
	}
}

type CursorDirection int

const (
	CursorUp CursorDirection = iota + 1
	CursorDown
)

func (t *TodoList) moveCursor(direction CursorDirection) {
	if len(t.items) == 0 {
		return
	}

	switch direction {
	case CursorUp:
		if t.selectedIndex > 0 {
			t.selectedIndex -= 1
		} else {
			t.selectedIndex = len(t.items) - 1
		}
	case CursorDown:
		if t.selectedIndex < len(t.items)-1 {
			t.selectedIndex += 1
		} else {
			t.selectedIndex = 0
		}
	}
}

func (t *TodoList) enterInputMode(action InputAction, initialValue string) {
	t.currentMode = ModeInput
	t.input = InputContext{
		Action:     action,
		Content:    initialValue,
		InitialVal: initialValue,
		Cursor:     len(initialValue),
	}
}

func (t *TodoList) AddItem(title string) error {
	if err := validateItemTitle(title); err != nil {
		return err
	}
	t.items = append(t.items, TodoItem{Title: title})
	return nil
}

func (t *TodoList) DeleteItem(index int) error {
	if !t.isValidIndex(index) {
		return &ValidationError{Operation: "delete", Err: errors.New("invalid index")}
	}
	t.items = slices.Delete(t.items, index, index+1)
	t.adjustCursorAfterDelete()
	return nil
}

func (t *TodoList) ToggleItem(index int) error {
	if !t.isValidIndex(index) {
		return &ValidationError{Operation: "toggle", Err: errors.New("invalid index")}
	}
	t.items[index].Completed = !t.items[index].Completed
	return nil
}

func (t *TodoList) ToggleAllItems() {
	allCompleted := true
	for _, item := range t.items {
		if !item.Completed {
			allCompleted = false
			break
		}
	}

	for i := range t.items {
		t.items[i].Completed = !allCompleted
	}
}

func (t TodoList) handleTextInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		t.handleInputSubmission()

	case tea.KeyEscape:
		t.exitInputMode()

	case tea.KeyBackspace:
		t.handleBackSpace()
		return t, nil

	case tea.KeySpace:
		t.insertAtCursor(" ")
	case tea.KeyRunes:
		t.insertAtCursor(string(msg.Runes))

	case tea.KeyLeft:
		if t.input.Cursor > 0 {
			t.input.Cursor--
		}

	case tea.KeyRight:
		if t.input.Cursor < len(t.input.Content) {
			t.input.Cursor++
		}

	case tea.KeyCtrlA, tea.KeyHome:
		t.input.Cursor = 0

	case tea.KeyCtrlE, tea.KeyEnd:
		t.input.Cursor = len(t.input.Content)
	}
	return t, nil
}

func (t *TodoList) handleInputSubmission() (tea.Model, tea.Cmd) {
	trimmedText := strings.TrimSpace(t.input.Content)

	if trimmedText == "" {
		return t, nil
	}
	if t.input.Action == ActionCreate {
		if err := t.AddItem(trimmedText); err != nil {
			t.lastErr = err
			return t, nil
		}
	}
	if t.input.Action == ActionEdit {
		t.items[t.selectedIndex].Title = trimmedText
	}

	t.exitInputMode()
	return t, nil
}

func (t *TodoList) insertAtCursor(text string) {
	t.input.Content = t.input.Content[:t.input.Cursor] + text + t.input.Content[t.input.Cursor:]
	t.input.Cursor += len(text)
}

func (t *TodoList) handleBackSpace() {
	if len(t.input.Content) > 0 && t.input.Cursor > 0 {
		t.input.Content = t.input.Content[:t.input.Cursor-1] + t.input.Content[t.input.Cursor:]
		t.input.Cursor--
	}
}

func (t *TodoList) exitInputMode() {
	t.currentMode = ModeNormal
	t.input = InputContext{}
}

// Bubble Tea

func (t TodoList) Init() tea.Cmd {
	return nil
}

func (t TodoList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		t.lastErr = msg
		return t, nil

	case tea.KeyMsg:
		switch t.currentMode {
		case ModeInput:
			return t.handleTextInputMode(msg)
		default:
			return t.handleNormalMode(msg)
		}
	}
	return t, nil
}

func (t TodoList) View() string {
	if t.lastErr != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.\n", t.lastErr)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("you have %d items on your list:\n\n", len(t.items)))

	for i, item := range t.items {
		cursor := " "
		if t.selectedIndex == i {
			cursor = ">"
		}

		checked := " "
		if item.Completed {
			checked = "x"
		}

		sb.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, item.Title))
	}

	sb.WriteString("\n")

	if t.currentMode == ModeInput {
		actionText := "edit item"
		if t.input.Action == ActionCreate {
			actionText = "enter new item"
		}
		sb.WriteString(fmt.Sprintf("%s (esc to cancel):\n", actionText))
		sb.WriteString(fmt.Sprintf("%s %s|%s\n", ">", t.input.Content[:t.input.Cursor], t.input.Content[t.input.Cursor:]))
	}
	if t.currentMode == ModeNormal {
		sb.WriteString("up/down: move cursor, enter/space: toggle, a: toggle all, n: new item, e: edit, d: delete, q/esc: quit")
	}

	return sb.String()
}

func main() {
	initialItems := []string{
		"Warm eba",
		"Exercise",
		"Read a book",
		"Write some slop",
	}

	p := tea.NewProgram(NewTodoList(initialItems), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running programme: %v\n", err)
		os.Exit(1)
	}
}
