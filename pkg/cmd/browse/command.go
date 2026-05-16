package browse

import (
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
)

type CommandHandler func(m *Model, args []string) (statusMsg string, err error)

type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     CommandHandler
}

type CommandRegistry struct {
	commands map[string]Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]Command),
	}
}

func (r *CommandRegistry) Register(cmd Command) {
	r.commands[cmd.Name] = cmd
}

func (r *CommandRegistry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

func (r *CommandRegistry) Complete(prefix string) []string {
	var matches []string
	for name := range r.commands {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches
}

func (r *CommandRegistry) All() []Command {
	var cmds []Command
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})
	return cmds
}

// ParseCommand splits input into command name and arguments.
func ParseCommand(input string) (name string, args []string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}
	parts := strings.Fields(input)
	return parts[0], parts[1:]
}

// pendingFetchCmd stores the parameters for a deferred fetch operation
type pendingFetchCmd struct {
	path      string
	isDoc     bool
	colIndex  int
	sortField string
	sortDir   firestore.Direction
}

func initCommandRegistry() *CommandRegistry {
	r := NewCommandRegistry()

	r.Register(Command{
		Name:        "help",
		Description: "List all available commands",
		Usage:       ":help",
		Handler:     cmdHelp,
	})

	r.Register(Command{
		Name:        "goto",
		Description: "Navigate to a Firestore path",
		Usage:       ":goto <path>",
		Handler:     cmdGoto,
	})

	r.Register(Command{
		Name:        "refresh",
		Description: "Refresh the current column",
		Usage:       ":refresh",
		Handler:     cmdRefresh,
	})

	r.Register(Command{
		Name:        "sort",
		Description: "Sort collection by field",
		Usage:       ":sort <field> [asc|desc]",
		Handler:     cmdSort,
	})

	return r
}

func cmdHelp(m *Model, args []string) (string, error) {
	var lines []string
	for _, cmd := range m.commandRegistry.All() {
		lines = append(lines, fmt.Sprintf("  %-20s %s", cmd.Usage, cmd.Description))
	}
	return "Commands:\n" + strings.Join(lines, "\n"), nil
}

func cmdGoto(m *Model, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: :goto <path>")
	}

	path := strings.Trim(strings.Join(args, " "), "/")
	isDoc := false
	if path != "" {
		segments := strings.Split(path, "/")
		isDoc = len(segments)%2 == 0
	}

	m.columns = []Column{{
		path:         path,
		isDoc:        isDoc,
		scrollOffset: 0,
	}}
	m.activeColumn = 0
	m.loading = true

	sortField, sortDir := m.getSortParams(path)
	m.pendingFetch = &pendingFetchCmd{path: path, isDoc: isDoc, colIndex: 0, sortField: sortField, sortDir: sortDir}

	return fmt.Sprintf("Navigating to /%s", path), nil
}

func cmdRefresh(m *Model, args []string) (string, error) {
	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no column to refresh")
	}

	col := m.columns[m.activeColumn]
	m.loading = true

	sortField, sortDir := m.getSortParams(col.path)
	m.pendingFetch = &pendingFetchCmd{path: col.path, isDoc: col.isDoc, colIndex: m.activeColumn, sortField: sortField, sortDir: sortDir}

	return "Refreshing...", nil
}

func cmdSort(m *Model, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: :sort <field> [asc|desc]")
	}

	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no column to sort")
	}

	col := m.columns[m.activeColumn]
	if col.isDoc {
		return "", fmt.Errorf("can only sort collections, not documents")
	}

	field := args[0]
	dir := firestore.Asc
	if len(args) > 1 {
		switch strings.ToLower(args[1]) {
		case "desc", "descending":
			dir = firestore.Desc
		case "asc", "ascending":
			dir = firestore.Asc
		default:
			return "", fmt.Errorf("invalid direction %q, use asc or desc", args[1])
		}
	}

	m.sortState[col.path] = sortStateEntry{
		Field:     field,
		Direction: dir,
	}

	m.loading = true
	m.pendingFetch = &pendingFetchCmd{path: col.path, isDoc: col.isDoc, colIndex: m.activeColumn, sortField: field, sortDir: dir}

	dirStr := "Ascending"
	if dir == firestore.Desc {
		dirStr = "Descending"
	}
	return fmt.Sprintf("Sorted by %s (%s)", field, dirStr), nil
}
