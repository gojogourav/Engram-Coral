package diff

import (
	"fmt"
	"strings"
	"unicode"
)

func normalizeForMatch(s string) string {
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u2018', '\u2019', '\u02BC', '\u02B9':
			b.WriteByte('\'')
		case '\u201C', '\u201D':
			b.WriteByte('"')
		case '\u2013', '\u2014':
			b.WriteByte('-')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func linesMatch(a, b string) bool {
	return normalizeForMatch(a) == normalizeForMatch(b)
}

// bestAnchor picks the most unique (longest non-blank) context or removed
// line from the hunk to use as the search anchor.
// Blank lines are terrible anchors — they appear everywhere in Go files.
func bestAnchor(hunk Hunk) (anchorContent string, anchorOffset int, ok bool) {
	best := ""
	bestOffset := 0
	oldLineOffset := 0 // tracks position relative to OldStart

	for _, l := range hunk.Lines {
		if l.Type == Added {
			continue // added lines don't exist in the original file
		}
		content := strings.TrimSpace(l.Content)
		if len(content) > len(strings.TrimSpace(best)) {
			best = l.Content
			bestOffset = oldLineOffset
		}
		oldLineOffset++
	}

	if strings.TrimSpace(best) == "" {
		return "", 0, false
	}
	return best, bestOffset, true
}

// findHunkStart searches for the real position of a hunk when the AI's
// stated line number is wrong. It uses the longest non-blank context/removed
// line as the anchor, then walks back by anchorOffset to get the true start.
func findHunkStart(originalLines []string, hunk Hunk, statedStart int) (int, error) {
	anchor, anchorOffset, ok := bestAnchor(hunk)
	if !ok {
		// Only added lines — no anchor possible, trust stated position
		return statedStart, nil
	}

	const searchRadius = 15

	low := statedStart - searchRadius
	if low < 1 {
		low = 1
	}
	high := statedStart + searchRadius
	if high > len(originalLines) {
		high = len(originalLines)
	}

	// Search outward from stated position
	for delta := 0; delta <= searchRadius; delta++ {
		for _, candidate := range []int{statedStart + delta, statedStart - delta} {
			if delta == 0 && candidate != statedStart {
				continue
			}
			if candidate < low || candidate > high {
				continue
			}
			if linesMatch(originalLines[candidate-1], anchor) {
				// candidate is where the anchor line lives.
				// Walk back by anchorOffset to get the true hunk start.
				realStart := candidate - anchorOffset
				if realStart < 1 {
					continue
				}
				if delta > 0 {
					fmt.Printf("WARNING: hunk line number corrected: stated %d, actual %d (off by %+d)\n",
						statedStart, realStart, realStart-statedStart)
				}
				return realStart, nil
			}
		}
	}

	return 0, fmt.Errorf(
		"could not locate hunk anchor %q near line %d (searched +/-%d lines)",
		anchor, statedStart, searchRadius,
	)
}

func applyHunks(original string, hunks []Hunk) (string, error) {
	originalLines := strings.Split(original, "\n")
	var outputLines []string
	cursor := 1

	for hunkIdx, hunk := range hunks {
		realStart, err := findHunkStart(originalLines, hunk, hunk.OldStart)
		if err != nil {
			return "", fmt.Errorf("hunk %d: %w", hunkIdx+1, err)
		}

		if realStart < cursor {
			return "", fmt.Errorf("overlapping hunks at line %d (cursor is at %d)", realStart, cursor)
		}
		if realStart > len(originalLines)+1 {
			return "", fmt.Errorf("hunk OldStart %d is beyond file length %d", realStart, len(originalLines))
		}

		// Copy unchanged lines before this hunk
		for cursor < realStart {
			outputLines = append(outputLines, originalLines[cursor-1])
			cursor++
		}

		for _, line := range hunk.Lines {
			switch line.Type {
			case Context:
				if cursor > len(originalLines) {
					return "", fmt.Errorf(
						"context line %q at cursor %d is beyond end of file (%d lines)",
						line.Content, cursor, len(originalLines),
					)
				}
				got := originalLines[cursor-1]
				if !linesMatch(got, line.Content) {
					return "", fmt.Errorf(
						"context mismatch at line %d:\n  expected: %q\n  got:      %q",
						cursor, line.Content, got,
					)
				}
				outputLines = append(outputLines, got)
				cursor++

			case Added:
				outputLines = append(outputLines, line.Content)

			case Removed:
				if cursor > len(originalLines) {
					return "", fmt.Errorf(
						"remove line %q at cursor %d is beyond end of file (%d lines)",
						line.Content, cursor, len(originalLines),
					)
				}
				got := originalLines[cursor-1]
				if !linesMatch(got, line.Content) {
					return "", fmt.Errorf(
						"remove mismatch at line %d:\n  expected: %q\n  got:      %q",
						cursor, line.Content, got,
					)
				}
				cursor++
			}
		}
	}

	for cursor <= len(originalLines) {
		outputLines = append(outputLines, originalLines[cursor-1])
		cursor++
	}

	return strings.Join(outputLines, "\n"), nil
}

func ApplyDiff(fileDiffs []FileDiff, fileContext map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for _, fd := range fileDiffs {
		original, ok := fileContext[fd.Path]
		if !ok {
			return nil, fmt.Errorf("file not found in context: %s", fd.Path)
		}
		patched, err := applyHunks(original, fd.Hunks)
		if err != nil {
			return nil, fmt.Errorf("failed to apply hunks to %s: %w", fd.Path, err)
		}
		result[fd.Path] = patched
	}
	return result, nil
}
