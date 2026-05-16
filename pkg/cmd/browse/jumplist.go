package browse

type JumpEntry struct {
	Path  string
	IsDoc bool
}

type Jumplist struct {
	entries []JumpEntry
	cursor  int
}

func NewJumplist() *Jumplist {
	return &Jumplist{cursor: -1}
}

func (j *Jumplist) Push(path string, isDoc bool) {
	// Truncate forward history
	if j.cursor < len(j.entries)-1 {
		j.entries = j.entries[:j.cursor+1]
	}
	j.entries = append(j.entries, JumpEntry{Path: path, IsDoc: isDoc})
	j.cursor = len(j.entries) - 1
}

func (j *Jumplist) Back() (JumpEntry, bool) {
	if j.cursor <= 0 {
		return JumpEntry{}, false
	}
	j.cursor--
	return j.entries[j.cursor], true
}

func (j *Jumplist) Forward() (JumpEntry, bool) {
	if j.cursor >= len(j.entries)-1 {
		return JumpEntry{}, false
	}
	j.cursor++
	return j.entries[j.cursor], true
}

func (j *Jumplist) Len() int {
	return len(j.entries)
}

func (j *Jumplist) Cursor() int {
	return j.cursor
}
