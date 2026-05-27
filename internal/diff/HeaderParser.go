package diff

import (
	"fmt"
	"strings"
)

func parseHunkHeader(line string) (Hunk, error) {
	var hunk Hunk

	if len(line) < 3 {
		return hunk, fmt.Errorf("malformed hunk header: %s", line)
	}

	inner := line[3:]
	end := strings.Index(inner, " @@")
	if end == -1 {
		end = strings.Index(inner, "@@")
		if end == -1 {
			return hunk, fmt.Errorf("malformed hunk header: %s", line)
		}
	}
	inner = strings.TrimSpace(inner[:end]) // "-5,7 +5,7"

	parts := strings.Fields(inner)
	if len(parts) != 2 {
		return hunk, fmt.Errorf("unexpected hunk header format: %s", line)
	}

	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "-"):
			// parse old range
			_, err := fmt.Sscanf(part, "-%d,%d", &hunk.OldStart, &hunk.OldCount)
			if err != nil {
				_, err = fmt.Sscanf(part, "-%d", &hunk.OldStart)
				if err != nil {
					return hunk, fmt.Errorf("failed to parse old range %q: %w", part, err)
				}
				hunk.OldCount = 1
			}

		case strings.HasPrefix(part, "+"):
			// parse new range
			_, err := fmt.Sscanf(part, "+%d,%d", &hunk.NewStart, &hunk.NewCount)
			if err != nil {
				_, err = fmt.Sscanf(part, "+%d", &hunk.NewStart)
				if err != nil {
					return hunk, fmt.Errorf("failed to parse new range %q: %w", part, err)
				}
				hunk.NewCount = 1
			}

		default:
			// Case 2 — no +/- prefix at all, e.g. "0,7"
			// Treat it as the old range since it appeared where - was expected
			_, err := fmt.Sscanf(part, "%d,%d", &hunk.OldStart, &hunk.OldCount)
			if err != nil {
				_, err = fmt.Sscanf(part, "%d", &hunk.OldStart)
				if err != nil {
					return hunk, fmt.Errorf("unrecognized hunk range %q: %w", part, err)
				}
				hunk.OldCount = 1
			}
		}
	}

	if hunk.OldStart == 0 && hunk.OldCount == 0 {
		hunk.OldStart = 1
		hunk.OldCount = hunk.NewCount
	}

	return hunk, nil
}
