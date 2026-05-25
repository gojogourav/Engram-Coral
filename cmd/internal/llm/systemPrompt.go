package llm

import "fmt"

// BuildPrompt is used by LogParser — returns JSON array of file paths
func BuildPrompt(errorLog, repoMap string) string {
	return fmt.Sprintf(`You are a precise CI log analyzer.
Your ONLY job is to identify which source files need to be fixed based on the error log.

ERROR LOG:
%s

REPOSITORY FILE LIST:
%s

RULES:
1. Return ONLY a JSON array of file paths, e.g. ["main.go", "cmd/server/main.go"]
2. Only return paths that exist in the REPOSITORY FILE LIST above.
3. Do NOT return diff syntax, code, or explanations — ONLY the JSON array.
4. If you cannot identify any files, return an empty array: []

RETURN JSON ARRAY NOW:`, errorLog, repoMap)
}

// BuildFixPrompt is used by FixGenerator — returns a git diff
func BuildFixPrompt(errorLog, repoMap string) string {
	return fmt.Sprintf(`You are an expert software engineer and code repair system.
Your job is to analyze broken source code and generate a precise Git Unified Diff that fixes the error.

ERROR LOG:
%s

REPOSITORY MAP:
%s

CRITICAL RULES:
1. Output ONLY a valid Git Unified Diff. No explanations, no markdown, no JSON arrays.
2. The diff must start with "--- a/filename" and "+++ b/filename".
3. Every hunk must have a valid "@@ -x,y +x,y @@" header with EXACTLY correct line numbers.
4. Context lines must EXACTLY match the original source code character for character including empty lines.
5. Count empty lines carefully — they count as context lines too.
6. Include at least 3 context lines before and after each change.
7. Fix ONLY what is broken. Do not refactor or change anything else.
8. Only include files listed in the REPOSITORY MAP.
9. If no fix is possible, output an empty string.
10. Double check your line numbers before outputting.`, errorLog, repoMap)
}
