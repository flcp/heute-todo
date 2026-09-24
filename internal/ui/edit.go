package ui

import (
	"strings"
	"time"

	"github.com/flcp/heute-todo/internal/todotxt"
)

// token is a whitespace-delimited chunk of the edit buffer, carrying its rune
// span [start,end) into the buffer so cursor positions can be computed.
type token struct {
	start, end int
	text       string
}

// scanTokens splits value into whitespace-delimited tokens, recording each
// token's rune span into value. A double-quoted value in a key:"..." tag is
// kept together (spaces included) so fields like details:"lorem ipsum" tokenize
// as one unit, mirroring todotxt.Fields.
func scanTokens(value string) []token {
	runes := []rune(value)
	var toks []token
	i := 0
	for i < len(runes) {
		for i < len(runes) && runes[i] == ' ' {
			i++
		}
		if i >= len(runes) {
			break
		}
		start := i
		for i < len(runes) && runes[i] != ' ' {
			if runes[i] == '"' && i > start && runes[i-1] == ':' {
				i++ // opening quote
				for i < len(runes) && runes[i] != '"' {
					i++
				}
				if i < len(runes) {
					i++ // closing quote
				}
				break
			}
			i++
		}
		toks = append(toks, token{start: start, end: i, text: string(runes[start:i])})
	}
	return toks
}

// bufferInfo records the token index of each recognized field in an edit
// buffer (-1 when absent). descStart is the token index where the description
// begins, after the leading completion marker, priority and dates.
type bufferInfo struct {
	toks                                            []token
	priority, project, context, due, details, title int
	descStart                                       int
}

// classifyBuffer locates each field within a raw todo.txt edit buffer. The
// leading prefix (optional "x", "(X)" priority and up to two dates) is skipped;
// remaining tokens are classified the same way the view splits a description
// (see splitDescription) and the todotxt tag rules (see splitTag).
func classifyBuffer(value string) bufferInfo {
	return bufferInfoFromTokens(scanTokens(value))
}

// isPriorityToken reports whether t is a "(X)" priority marker with X in A–Z.
func isPriorityToken(t string) bool {
	return len(t) == 3 && t[0] == '(' && t[2] == ')' && t[1] >= 'A' && t[1] <= 'Z'
}

// isDateToken reports whether t is a YYYY-MM-DD date.
func isDateToken(t string) bool {
	if len(t) != 10 {
		return false
	}
	_, err := time.Parse(todotxt.DateLayout, t)
	return err == nil
}

// isKeyValue reports whether t is a "key:value" tag (non-empty key and value),
// excluding URLs whose value part starts with "//". This mirrors the metadata
// test in splitDescription.
func isKeyValue(t string) bool {
	i := strings.IndexByte(t, ':')
	if i <= 0 || i >= len(t)-1 {
		return false
	}
	if strings.HasPrefix(t[i+1:], "//") {
		return false
	}
	return true
}

// locateField returns the rune index where field f's value begins in value, and
// whether the field is currently present. For a missing field the returned
// index is a sensible fallback (description start for the title, end of buffer
// otherwise) and present is false.
func locateField(value string, f editField) (cursorPos int, present bool) {
	info := classifyBuffer(value)
	runes := []rune(value)
	end := len(runes)

	switch f {
	case fieldTitle:
		if info.title != -1 {
			return info.toks[info.title].start, true
		}
		if info.descStart < len(info.toks) {
			return info.toks[info.descStart].start, false
		}
		return end, false
	case fieldPriority:
		if info.priority != -1 {
			return info.toks[info.priority].start + 1, true // the letter
		}
	case fieldProject:
		if info.project != -1 {
			return info.toks[info.project].start + 1, true // after '+'
		}
	case fieldContext:
		if info.context != -1 {
			return info.toks[info.context].start + 1, true // after '@'
		}
	case fieldDue:
		if info.due != -1 {
			return info.toks[info.due].start + len("due:"), true // after "due:"
		}
	case fieldDetails:
		if info.details != -1 {
			return info.toks[info.details].start + len(`details:"`), true // inside the quotes
		}
	}
	return end, false
}

// ensureField returns value (possibly with an inserted empty scaffold for a
// missing field) and the rune index at which the field's value should be
// edited. The title is never scaffolded.
func ensureField(value string, f editField) (newValue string, cursorPos int) {
	if pos, ok := locateField(value, f); ok {
		return value, pos
	}
	switch f {
	case fieldPriority:
		return scaffoldPriority(value)
	case fieldProject:
		return appendScaffold(value, "+")
	case fieldContext:
		return appendScaffold(value, "@")
	case fieldDue:
		return appendScaffold(value, "due:")
	case fieldDetails:
		return scaffoldDetails(value)
	}
	// Title (or any non-scaffolding field): keep value, use the fallback pos.
	pos, _ := locateField(value, f)
	return value, pos
}

// appendScaffold appends scaffold to the end of value (with a separating space
// when value is non-empty) and returns the cursor at the end, ready for the
// user to type the value.
func appendScaffold(value, scaffold string) (string, int) {
	base := strings.TrimRight(value, " ")
	if base == "" {
		return scaffold, len([]rune(scaffold))
	}
	nv := base + " " + scaffold
	return nv, len([]rune(nv))
}

// scaffoldPriority inserts an "()" priority after any leading completion
// marker and returns the cursor between the brackets.
func scaffoldPriority(value string) (string, int) {
	toks := scanTokens(value)
	runes := []rune(value)
	if len(toks) > 0 && toks[0].text == "x" {
		left := string(runes[:toks[0].end])
		right := string(runes[toks[0].end:])
		return left + " ()" + right, len([]rune(left)) + 2 // between the brackets
	}
	if strings.TrimSpace(value) == "" {
		return "()", 1
	}
	return "() " + value, 1
}

// scaffoldDetails appends an empty details:"" tag and returns the cursor
// between the quotes, ready for a (possibly multi-word) value.
func scaffoldDetails(value string) (string, int) {
	base := strings.TrimRight(value, " ")
	if base == "" {
		return `details:""`, len([]rune(`details:""`)) - 1
	}
	nv := base + ` details:""`
	return nv, len([]rune(nv)) - 1
}

// stripEmptyField removes field f's token from value (collapsing surrounding
// whitespace to single spaces) when that field is an empty scaffold: a bare
// "+", "@", "due:", details:"" or a malformed "()" priority. The title is never
// removed and a filled field is left untouched.
func stripEmptyField(value string, f editField) string {
	toks := scanTokens(value)
	idx := emptyFieldTokenIndex(toks, f)
	if idx == -1 {
		return value
	}
	kept := make([]string, 0, len(toks))
	for i, t := range toks {
		if i != idx {
			kept = append(kept, t.text)
		}
	}
	return strings.Join(kept, " ")
}

// emptyFieldTokenIndex returns the index of field f's token in toks when it is
// present but empty, or -1 otherwise.
func emptyFieldTokenIndex(toks []token, f editField) int {
	if f == fieldPriority {
		for i, t := range toks {
			if t.text == "()" {
				return i
			}
		}
		return -1
	}
	info := bufferInfoFromTokens(toks)
	switch f {
	case fieldProject:
		if info.project != -1 && toks[info.project].text == "+" {
			return info.project
		}
	case fieldContext:
		if info.context != -1 && toks[info.context].text == "@" {
			return info.context
		}
	case fieldDue:
		if info.due != -1 && toks[info.due].text == "due:" {
			return info.due
		}
	case fieldDetails:
		if info.details != -1 && toks[info.details].text == `details:""` {
			return info.details
		}
	}
	return -1
}

// bufferInfoFromTokens classifies an already-tokenized buffer, so callers with
// tokens in hand avoid re-scanning.
func bufferInfoFromTokens(toks []token) bufferInfo {
	info := bufferInfo{toks: toks, priority: -1, project: -1, context: -1, due: -1, details: -1, title: -1}
	i := 0
	if i < len(toks) && toks[i].text == "x" {
		i++
	}
	if i < len(toks) && isPriorityToken(toks[i].text) {
		info.priority = i
		i++
	}
	for d := 0; d < 2 && i < len(toks) && isDateToken(toks[i].text); d++ {
		i++
	}
	info.descStart = i
	for ; i < len(toks); i++ {
		t := toks[i].text
		switch {
		case len(t) >= 1 && t[0] == '+':
			if info.project == -1 {
				info.project = i
			}
		case len(t) >= 1 && t[0] == '@':
			if info.context == -1 {
				info.context = i
			}
		case strings.HasPrefix(t, "due:"):
			if info.due == -1 {
				info.due = i
			}
		case strings.HasPrefix(t, "details:"):
			if info.details == -1 {
				info.details = i
			}
		case isKeyValue(t):
		default:
			if info.title == -1 {
				info.title = i
			}
		}
	}
	return info
}
