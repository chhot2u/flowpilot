package batch

import "strings"

// DefaultNameTemplate returns the fallback naming template for batch tasks.
func DefaultNameTemplate() string {
	return "Task {{index}} - {{domain}}"
}

// ValidateTemplate checks that only supported variables are used.
func ValidateTemplate(template string) bool {
	allowed := []string{"{{url}}", "{{domain}}", "{{index}}", "{{name}}"}
	for strings.Contains(template, "{{") {
		start := strings.Index(template, "{{")
		end := strings.Index(template[start+2:], "}}")
		if end == -1 {
			return false
		}
		end = start + 2 + end
		expr := template[start : end+2]
		valid := false
		for _, a := range allowed {
			if expr == a {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
		template = template[end+2:]
	}
	return true
}
