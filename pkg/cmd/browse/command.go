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
		Name:        "quit",
		Description: "Quit the application",
		Usage:       ":quit",
		Handler:     cmdQuit,
	})

	r.Register(Command{
		Name:        "man",
		Description: "Show the full manual",
		Usage:       ":man",
		Handler:     cmdMan,
	})

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

	r.Register(Command{
		Name:        "add",
		Description: "Create a new document in the current collection",
		Usage:       ":add [id]",
		Handler:     cmdAdd,
	})

	r.Register(Command{
		Name:        "rename",
		Description: "Rename or move a document (copy to new path, delete old)",
		Usage:       ":rename <path>",
		Handler:     cmdRename,
	})

	r.Register(Command{
		Name:        "mv",
		Description: "Alias for :rename",
		Usage:       ":mv <path>",
		Handler:     cmdRename,
	})

	r.Register(Command{
		Name:        "copy",
		Description: "Copy a document to a new id (blank = auto-generate)",
		Usage:       ":copy [id|path]",
		Handler:     cmdCopy,
	})

	r.Register(Command{
		Name:        "cp",
		Description: "Alias for :copy",
		Usage:       ":cp [id|path]",
		Handler:     cmdCopy,
	})

	return r
}

func cmdMan(m *Model, args []string) (string, error) {
	return `FIRESTORE BROWSE — MANUAL

OVERVIEW
  A vim-style TUI for browsing Firestore databases.
  Navigate collections and documents using Miller columns.

MODES
  NORMAL    Default mode. Navigate, view, and operate on data.
  VISUAL    Select multiple items for bulk operations.
  COMMAND   Enter commands with : prefix.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
NAVIGATION

  j / ↓           Move cursor down
  k / ↑           Move cursor up
  g               Jump to top of column
  G               Jump to bottom of column
  l / → / Enter   Navigate forward (open collection/document)
  h / ← / Bksp    Navigate back (close column / collapse node)
  Space           Toggle expand/collapse on tree node

  Ctrl+d / PgDn   Scroll down (half page)
  Ctrl+u / PgUp   Scroll up (half page)

  Ctrl+g          Go to path (opens path input dialog)
  /               Filter current column (substring match)

  Ctrl+o          Jump back in history
  Ctrl+i          Jump forward in history

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MARKS & BOOKMARKS

  m + [a-z]       Set a mark at current location
  ' + [a-z]       Jump to a mark

  :marks          List all set marks

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TREE VIEW CONTROLS

  zM              Fold all (collapse entire tree)
  zR              Unfold all (expand entire tree)
  z1              Fold to depth 1
  z2              Fold to depth 2
  z3              Fold to depth 3

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
DOCUMENT OPERATIONS

  e               Edit document (opens $EDITOR)
  d               Delete document (with confirmation)
  R               Rename / move document (copy to new path, delete old)
  c               Copy document to a new id (prompts; blank = auto-generate)
  r               Refresh current column
  p               Toggle preview pane

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CLIPBOARD / COPY

  yy              Copy selected value (field value or path)
  ya              Copy entire document as JSON
  Y               Copy ID or field key

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SORT & QUERY

  s               Open sort dialog
  S               Clear sort for current collection

  :sort <f> [dir]       Sort by field (asc/desc)
  :query <f> <op> <v>   Server-side Firestore query
                        Operators: ==  !=  <  >  <=  >=
                                   array-contains  in

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
VISUAL MODE

  v               Toggle item selection / enter visual mode
  V               Range select (anchor + move to extend)
  j / k           Move and extend selection
  d               Bulk delete selected documents
  Esc             Exit visual mode

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
COMMANDS

  :help                   List available commands
  :man                    Show this manual
  :quit                   Quit the application
  :goto <path>            Navigate to a Firestore path
  :refresh                Refresh current column
  :sort <field> [asc|desc]  Sort collection by field
  :query <f> <op> <val>   Filter with a server-side query
  :set limit <n>          Set pagination limit (default: 50)
  :export json|ndjson [f] Export documents (to file or clipboard)
  :add [id]               Create a new document
  :rename <path>          Rename / move a document (bare name = same
                          collection; path with / = absolute from root)
  :mv <path>              Alias for :rename
  :copy [id|path]         Copy a document (no arg = auto id; bare name = same
                          collection; path with / = absolute from root)
  :cp [id|path]           Alias for :copy
  :marks                  List all bookmarks

  Tab in command mode autocompletes command names.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
QUITTING

  q               Quit
  :quit           Quit
  Ctrl+c          Quit (except in command mode, where it cancels)
  Esc             Shows "Press q to quit" hint`, nil
}

func cmdQuit(m *Model, args []string) (string, error) {
	m.pendingQuit = true
	return "", nil
}

func cmdHelp(m *Model, args []string) (string, error) {
	var lines []string
	for _, cmd := range m.commandRegistry.All() {
		lines = append(lines, fmt.Sprintf("  %-20s %s", cmd.Usage, cmd.Description))
	}
	return "Commands:\n" + strings.Join(lines, "\n") + "\n\nType :man for the full manual.", nil
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

func cmdAdd(m *Model, args []string) (string, error) {
	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no active column")
	}

	col := m.columns[m.activeColumn]
	if col.isDoc {
		return "", fmt.Errorf("can only add documents to collections")
	}

	docID := ""
	if len(args) > 0 {
		docID = args[0]
	}

	m.pendingEditor = startAddCmd(m.client, col.path, docID, col.availableFields, m.activeColumn)
	return "", nil
}

func cmdRename(m *Model, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: :rename <path>")
	}

	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no active column")
	}

	col := m.columns[m.activeColumn]
	var src string
	var fromDocView bool

	if col.isDoc {
		src = col.path
		fromDocView = true
	} else {
		item := m.getSelectedItem()
		if item == nil || !item.isDoc {
			return "", fmt.Errorf("select a document to rename")
		}
		src = item.path
		fromDocView = false
	}

	dst, err := resolveRenameTarget(src, strings.Join(args, " "))
	if err != nil {
		return "", err
	}

	m.renameSrc = src
	m.renameFromDocView = fromDocView
	m.loading = true
	m.pendingCmd = prepareRename(m.client, src, dst, fromDocView)
	return "", nil
}

func cmdCopy(m *Model, args []string) (string, error) {
	if m.activeColumn >= len(m.columns) {
		return "", fmt.Errorf("no active column")
	}

	col := m.columns[m.activeColumn]
	var src string

	if col.isDoc {
		src = col.path
	} else {
		item := m.getSelectedItem()
		if item == nil || !item.isDoc {
			return "", fmt.Errorf("select a document to copy")
		}
		src = item.path
	}

	dst := ""
	if input := strings.TrimSpace(strings.Join(args, " ")); input != "" {
		resolved, err := resolveRenameTarget(src, input)
		if err != nil {
			return "", err
		}
		dst = resolved
	}

	m.copySrc = src
	m.loading = true
	m.pendingCmd = executeCopy(m.client, src, dst)
	return "", nil
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
