package clone

import "sync/atomic"

// stats are the live counters of a run, read by the CLI's progress ticker.
type stats struct {
	pages       atomic.Int64
	assets      atomic.Int64
	pageErrors  atomic.Int64
	assetErrors atomic.Int64
	skipped     atomic.Int64 // robots-disallowed or out of budget
}

// Progress is a snapshot of a run for display.
type Progress struct {
	Pages       int64
	Assets      int64
	PageErrors  int64
	AssetErrors int64
	Skipped     int64
}

func (s *stats) snapshot() Progress {
	return Progress{
		Pages:       s.pages.Load(),
		Assets:      s.assets.Load(),
		PageErrors:  s.pageErrors.Load(),
		AssetErrors: s.assetErrors.Load(),
		Skipped:     s.skipped.Load(),
	}
}

// Result is the final outcome returned by Run.
type Result struct {
	Progress
	OutDir string
}
