package batch

import "web-automation/internal/models"

// DefaultNameTemplate returns the fallback naming template for batch tasks.
func DefaultNameTemplate() string {
	return "Task {{index}} - {{domain}}"
}

// ValidateTemplate checks that only supported variables are used.
func ValidateTemplate(template string) bool {
	return models.ValidateBatchTemplate(template)
}
