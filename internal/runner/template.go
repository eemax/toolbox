package runner

import (
	"fmt"
	"regexp"
	"strings"
)

var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

// ResolveTemplate resolves variables-only placeholders. Unknown variables are errors.
func ResolveTemplate(value string, vars map[string]string) (string, error) {
	if !strings.Contains(value, "{{") {
		return value, nil
	}
	indexes := variablePattern.FindAllStringSubmatchIndex(value, -1)
	if len(indexes) == 0 {
		return "", fmt.Errorf("invalid template syntax in %q", value)
	}
	var builder strings.Builder
	cursor := 0
	for _, index := range indexes {
		fullStart, fullEnd := index[0], index[1]
		keyStart, keyEnd := index[2], index[3]
		builder.WriteString(value[cursor:fullStart])
		key := value[keyStart:keyEnd]
		replacement, ok := vars[key]
		if !ok {
			return "", fmt.Errorf("unknown template variable %q", key)
		}
		builder.WriteString(replacement)
		cursor = fullEnd
	}
	builder.WriteString(value[cursor:])
	return builder.String(), nil
}

// ResolveSlice resolves templates in each slice item.
func ResolveSlice(values []string, vars map[string]string) ([]string, error) {
	resolved := make([]string, len(values))
	for i, value := range values {
		item, err := ResolveTemplate(value, vars)
		if err != nil {
			return nil, err
		}
		resolved[i] = item
	}
	return resolved, nil
}
