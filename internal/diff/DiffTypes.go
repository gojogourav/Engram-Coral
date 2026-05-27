package diff

type LineType int

const (
	Context LineType = iota //  space " " line — unchanged
	Added                   //  "+"
	Removed                 //  "-"
)

type Line struct {
	Type    LineType
	Content string
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
}

type FileDiff struct {
	Path  string // e.g. "cmd/internal/api/handler.go"
	Hunks []Hunk
}
