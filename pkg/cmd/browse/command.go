package browse

import (
	"fmt"
	"sort"
	"strconv"
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
	opts      []fetchOption
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

	r.Register(Command{
		Name:        "set",
		Description: "Set a configuration value",
		Usage:       ":set limit <n>",
		Handler:     cmdSet,
	})

	r.Register(Command{
		Name:        "export",
		Description: "Export documents as JSON or NDJSON",
		Usage:       ":export json|ndjson [file]",
		Handler:     cmdExport,
	})

	r.Register(Command{
		Name:        "marks",
		Description: "List all current marks",
		Usage:       ":marks",
		Handler:     cmdMarks,
	})

	r.Register(Command{
		Name:        "query",
		Description: "Filter collection with a Firestore query",
		Usage:       ":query <field> <op> <value>",
		Handler:     cmdQuery,
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

	// Push current location to jumplist
	if m.activeColumn < len(m.columns) {
		cur := m.columns[m.activeColumn]
		m.jumplist.Push(cur.path, cur.isDoc)
	}

	m.columns = []Column{{
		path:         path,
		isDoc:        isDoc,
		scrollOffset: 0,
	}}
	m.activeColumn = 0
	m.loading = true

	sortField, sortDir := m.getSortParams(path)
	m.pendingFetch = &pendingFetchCmd{path: path, isDoc: isDoc, colIndex: 0, sortField: sortField, sortDir: sortDir, opts: []fetchOption{withLimit(m.pageLimit)}}

	return fmt.Sprintf("Navigating to /%s", path), nil
}

func cmdRefresh(m *Model, args []string) (string, error) {
	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no column to refresh")
	}

	col := m.columns[m.activeColumn]
	m.columns[m.activeColumn].activeQuery = nil
	m.loading = true

	sortField, sortDir := m.getSortParams(col.path)
	m.pendingFetch = &pendingFetchCmd{path: col.path, isDoc: col.isDoc, colIndex: m.activeColumn, sortField: sortField, sortDir: sortDir, opts: []fetchOption{withLimit(m.pageLimit)}}

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
	m.pendingFetch = &pendingFetchCmd{path: col.path, isDoc: col.isDoc, colIndex: m.activeColumn, sortField: field, sortDir: dir, opts: []fetchOption{withLimit(m.pageLimit)}}

	dirStr := "Ascending"
	if dir == firestore.Desc {
		dirStr = "Descending"
	}
	return fmt.Sprintf("Sorted by %s (%s)", field, dirStr), nil
}

func cmdQuery(m *Model, args []string) (string, error) {
	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no active column")
	}

	col := m.columns[m.activeColumn]
	if col.isDoc {
		return "", fmt.Errorf("can only query collections, not documents")
	}

	q, err := ParseQueryArgs(args)
	if err != nil {
		return "", err
	}

	m.columns[m.activeColumn].activeQuery = &q
	m.loading = true

	sortField, sortDir := m.getSortParams(col.path)
	m.pendingFetch = &pendingFetchCmd{
		path: col.path, isDoc: col.isDoc, colIndex: m.activeColumn,
		sortField: sortField, sortDir: sortDir,
		opts: []fetchOption{withLimit(m.pageLimit), withQuery(&q)},
	}

	return fmt.Sprintf("Query: %s", q.String()), nil
}

func cmdMarks(m *Model, args []string) (string, error) {
	if len(m.marks) == 0 {
		return "No marks set", nil
	}

	// Sort marks by letter
	var letters []rune
	for k := range m.marks {
		letters = append(letters, k)
	}
	sort.Slice(letters, func(i, j int) bool { return letters[i] < letters[j] })

	var lines []string
	for _, letter := range letters {
		mark := m.marks[letter]
		lines = append(lines, fmt.Sprintf("  '%c'  %s", letter, mark.path))
	}
	return "Marks:\n" + strings.Join(lines, "\n"), nil
}

func cmdSet(m *Model, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: :set limit <n>")
	}

	switch args[0] {
	case "limit":
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			return "", fmt.Errorf("limit must be a positive integer")
		}
		m.pageLimit = n
		return fmt.Sprintf("Page limit set to %d", n), nil
	default:
		return "", fmt.Errorf("unknown setting: %s", args[0])
	}
}
