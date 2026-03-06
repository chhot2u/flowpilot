package models

// MaxBatchSize is the maximum number of URLs allowed in a single batch.
const MaxBatchSize = 10000

// AdvancedBatchInput holds the configuration for creating batch tasks
// from a recorded flow with shared steps.
type AdvancedBatchInput struct {
	FlowID         string      `json:"flowId"`
	URLs           []string    `json:"urls"`
	NamingTemplate string      `json:"namingTemplate"` // e.g. "{{index}} - {{domain}}"
	Priority       int         `json:"priority"`
	Proxy          ProxyConfig `json:"proxy"`
	Tags           []string    `json:"tags,omitempty"`
	AutoStart      bool        `json:"autoStart"`
}

// BatchGroup tracks a group of tasks created together from one batch operation.
type BatchGroup struct {
	ID      string   `json:"id"`
	FlowID  string   `json:"flowId"`
	TaskIDs []string `json:"taskIds"`
	Total   int      `json:"total"`
	Name    string   `json:"name"`
}

// BatchProgress reports aggregate execution status for a batch group.
type BatchProgress struct {
	BatchID   string `json:"batchId"`
	Total     int    `json:"total"`
	Pending   int    `json:"pending"`
	Queued    int    `json:"queued"`
	Running   int    `json:"running"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
	Cancelled int    `json:"cancelled"`
}

// TemplateVariable defines a supported substitution variable for batch naming
// and step value templates.
type TemplateVariable struct {
	Name        string `json:"name"`        // e.g. "url"
	Placeholder string `json:"placeholder"` // e.g. "{{url}}"
	Description string `json:"description"`
}

// SupportedVariables returns all template variables available for substitution.
func SupportedVariables() []TemplateVariable {
	return []TemplateVariable{
		{Name: "url", Placeholder: "{{url}}", Description: "Full URL of the task"},
		{Name: "domain", Placeholder: "{{domain}}", Description: "Domain extracted from URL"},
		{Name: "index", Placeholder: "{{index}}", Description: "1-based index in the batch"},
		{Name: "name", Placeholder: "{{name}}", Description: "Generated task name"},
	}
}
