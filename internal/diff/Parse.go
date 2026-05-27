package diff

import (
	"fmt"
	"strings"
)

// understand this code
func ParseDiff(rawDiff string) ([]FileDiff, error) {
	var fileDiffs []FileDiff
	var currentFile *FileDiff
	var currentHunk *Hunk

	lines := strings.Split(rawDiff, "\n")

	for i, line := range lines {
		// "--- a/main.go" signals a new file is starting
		if strings.HasPrefix(line, "--- a/") {
			path := strings.TrimPrefix(line, "--- a/")
			fileDiffs = append(fileDiffs, FileDiff{Path: path})
			currentFile = &fileDiffs[len(fileDiffs)-1]
			currentHunk = nil
			continue
		}

		// "+++ b/main.go" is redundant — path already captured above
		if strings.HasPrefix(line, "+++ b/") {
			continue
		}

		// Skip metadata lines
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") {
			continue
		}

		// "@@ -5,7 +5,7 @@" starts a new hunk
		if strings.HasPrefix(line, "@@") {
			if currentFile == nil {
				return nil, fmt.Errorf("hunk found before file header at line %d", i)
			}

			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse hunk header: %w", err)
			}

			currentFile.Hunks = append(currentFile.Hunks, hunk)
			currentHunk = &currentFile.Hunks[len(currentFile.Hunks)-1]
			continue

		}
		if currentHunk == nil {
			continue
		}

		if strings.HasPrefix(line, "+") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    Added,
				Content: line[1:],
			})
		} else if strings.HasPrefix(line, "-") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    Removed,
				Content: line[1:],
			})
		} else if strings.HasPrefix(line, " ") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    Context,
				Content: line[1:],
			})
		}

	}

	return fileDiffs, nil
}
