package modules

import (
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// BaseSchedulerModule provides common functionality for all scheduler modules
type BaseSchedulerModule struct {
	mu       sync.RWMutex
	logger   interfaces.Logger
	templates map[string]*template.Template
}

// NewBaseSchedulerModule creates a new base scheduler module
func NewBaseSchedulerModule() *BaseSchedulerModule {
	return &BaseSchedulerModule{
		logger:    nil, // Will be set by parent
		templates: make(map[string]*template.Template),
	}
}

// GetLogger returns the module's logger
func (b *BaseSchedulerModule) SetLogger(logger interfaces.Logger) {
	b.logger = logger
}

// GetLogger returns the module's logger
func (b *BaseSchedulerModule) GetLogger() interfaces.Logger {
	return b.logger
}

// BuildPrompt builds a prompt from a template with given parameters
func (b *BaseSchedulerModule) BuildPrompt(templateName string, params map[string]interface{}) (string, error) {
	b.mu.RLock()
	tmpl, exists := b.templates[templateName]
	b.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template '%s' not found", templateName)
	}

	var builder strings.Builder
	err := tmpl.Execute(&builder, params)
	if err != nil {
		return "", fmt.Errorf("failed to execute template '%s': %w", templateName, err)
	}

	return builder.String(), nil
}

// RegisterTemplate registers a template for prompt building
func (b *BaseSchedulerModule) RegisterTemplate(name, templateStr string) error {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template '%s': %w", name, err)
	}

	b.mu.Lock()
	b.templates[name] = tmpl
	b.mu.Unlock()

	return nil
}

// PromptTemplates defines common prompt templates used across scheduler modules
var PromptTemplates = map[string]string{
	"intent_recognizing": `
You are an intelligent assistant helping to determine whether new information retrieval is needed.

Current working memory:
{{range .working_memory_list}}- {{.}}
{{end}}

User queries:
{{range .q_list}}- {{.}}
{{end}}

Based on the user queries and current working memory, determine:
1. Whether retrieval is needed (trigger_retrieval: true/false)
2. What missing evidence needs to be retrieved (missing_evidence: ["item1", "item2"])
3. Confidence level (0.0 to 1.0)

Respond in JSON format:
{
  "trigger_retrieval": boolean,
  "missing_evidence": ["evidence1", "evidence2"],
  "confidence": 0.95
}
`,

	"memory_reranking": `
You are tasked with reranking memory items based on relevance and importance.

Query: {{.query}}

Current memory order:
{{range $i, $mem := .current_order}}{{$i}}: {{$mem}}
{{end}}

Staging buffer:
{{range $i, $mem := .staging_buffer}}{{$i}}: {{$mem}}
{{end}}

Rerank all memories by relevance and return the new order as a JSON list.
Respond in JSON format:
{
  "new_order": ["memory1", "memory2", "memory3"]
}
`,

	"freq_detecting": `
Analyze which memories from the activation memory list appear in the given answer.
For each memory that appears, increment its count by 1.

Answer: {{.answer}}

Activation memory list:
{{range .activation_memory_freq_list}}- Memory: "{{.memory}}", Count: {{.count}}
{{end}}

Return the updated list in JSON format:
{
  "activation_memory_freq_list": [
    {"memory": "text", "count": updated_count},
    ...
  ]
}
`,

	"memory_assembly": `
Assembled Memory Context:

{{.memory_text}}

This represents the current active memory context for processing.
`,
}

// InitializeDefaultTemplates registers all default prompt templates
func (b *BaseSchedulerModule) InitializeDefaultTemplates() error {
	for name, templateStr := range PromptTemplates {
		if err := b.RegisterTemplate(name, templateStr); err != nil {
			return fmt.Errorf("failed to register template '%s': %w", name, err)
		}
	}
	return nil
}