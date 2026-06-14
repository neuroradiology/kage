package clone

import (
	"path/filepath"
	"testing"
)

func TestFrontierOfferDedups(t *testing.T) {
	f := newFrontier()
	if !f.offer("a") {
		t.Fatal("first offer of a should be new")
	}
	if f.offer("a") {
		t.Fatal("second offer of a should be a duplicate")
	}
	if !f.offer("b") {
		t.Fatal("first offer of b should be new")
	}
}

func TestFrontierVisited(t *testing.T) {
	f := newFrontier()
	if f.isVisited("x") {
		t.Fatal("x should not be visited yet")
	}
	f.markVisited("x")
	if !f.isVisited("x") {
		t.Fatal("x should be visited after markVisited")
	}
	if f.visitedCount() != 1 {
		t.Fatalf("visitedCount = %d, want 1", f.visitedCount())
	}
}

func TestFrontierSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "state.json")

	f := newFrontier()
	f.markVisited("https://example.com/")
	f.markVisited("https://example.com/about")
	if err := f.save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	g := newFrontier()
	if err := g.load(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	if g.visitedCount() != 2 {
		t.Fatalf("loaded visitedCount = %d, want 2", g.visitedCount())
	}
	if !g.isVisited("https://example.com/about") {
		t.Fatal("about should be visited after load")
	}
	// A loaded visited URL is also seen, so it is not re-offered.
	if g.offer("https://example.com/about") {
		t.Fatal("a loaded URL should not be offered again")
	}
}

func TestFrontierLoadMissingIsNotError(t *testing.T) {
	f := newFrontier()
	if err := f.load(filepath.Join(t.TempDir(), "nope.json")); err != nil {
		t.Fatalf("loading a missing file should be a no-op, got %v", err)
	}
}
