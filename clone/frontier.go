package clone

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// frontier is the deduped set of page URLs kage has already seen. It is small,
// concurrency-safe, and persists to disk so --resume can skip work already done.
// The actual queueing is handled by the cloner's channels; the frontier only
// answers "is this URL new?" and remembers the answer.
type frontier struct {
	mu      sync.Mutex
	seen    map[string]bool // queued or visited
	visited map[string]bool // fully written
}

func newFrontier() *frontier {
	return &frontier{seen: map[string]bool{}, visited: map[string]bool{}}
}

// offer reports whether key is new (and records it as seen). A repeated key
// returns false so it is enqueued only once.
func (f *frontier) offer(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen[key] {
		return false
	}
	f.seen[key] = true
	return true
}

// markVisited records that a page was written.
func (f *frontier) markVisited(key string) {
	f.mu.Lock()
	f.visited[key] = true
	f.mu.Unlock()
}

// isVisited reports whether a page was already written in a previous run.
func (f *frontier) isVisited(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.visited[key]
}

func (f *frontier) visitedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.visited)
}

// state is the JSON shape persisted for resume.
type state struct {
	Visited []string `json:"visited"`
}

// load reads a previously saved visited set; a missing file is not an error.
func (f *frontier) load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range s.Visited {
		f.visited[v] = true
		f.seen[v] = true
	}
	return nil
}

// save writes the visited set atomically (write temp, rename).
func (f *frontier) save(path string) error {
	f.mu.Lock()
	visited := make([]string, 0, len(f.visited))
	for v := range f.visited {
		visited = append(visited, v)
	}
	f.mu.Unlock()
	sort.Strings(visited)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state{Visited: visited}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
