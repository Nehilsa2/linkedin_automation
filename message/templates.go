package message

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const TemplatesFile = "message_templates.json"

// TemplateManager manages message templates
type TemplateManager struct {
	Templates []Template `json:"templates"`
}

// DefaultTemplates returns built-in templates
func DefaultTemplates() []Template {
	return []Template{
		{
			Name:        "follow_up_generic",
			Description: "Generic follow-up message for new connections",
			Content:     "Hi {name}! Thanks for connecting. I noticed you're working at {company} - would love to learn more about your work. Looking forward to staying in touch!",
			Variables:   []string{"{name}", "{company}"},
		},
		{
			Name:        "follow_up_software",
			Description: "Follow-up for software engineers",
			Content:     "Hi {name}! Great to connect with a fellow developer. I see you're at {company} - always interested to hear about the tech stack and challenges teams are tackling. What are you working on these days?",
			Variables:   []string{"{name}", "{company}"},
		},
		{
			Name:        "follow_up_recruiter",
			Description: "Follow-up for recruiters/HR",
			Content:     "Hi {name}, thanks for connecting! I'm always open to hearing about interesting opportunities. Feel free to reach out if you come across anything that might be a good fit.",
			Variables:   []string{"{name}"},
		},
		{
			Name:        "follow_up_founder",
			Description: "Follow-up for founders/entrepreneurs",
			Content:     "Hi {name}! Excited to connect. I saw you're building something at {company} - would love to hear more about your journey. Always inspiring to connect with fellow builders!",
			Variables:   []string{"{name}", "{company}"},
		},
		{
			Name:        "follow_up_simple",
			Description: "Simple thank you message",
			Content:     "Hi {name}! Thanks for accepting my connection request. Looking forward to staying in touch!",
			Variables:   []string{"{name}"},
		},
	}
}

// LoadTemplates loads templates from file or returns defaults
func LoadTemplates() (*TemplateManager, error) {
	manager := &TemplateManager{
		Templates: DefaultTemplates(),
	}

	data, err := os.ReadFile(TemplatesFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Save default templates
			manager.Save()
			return manager, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, manager); err != nil {
		return nil, err
	}

	return manager, nil
}

// Save saves templates to file
func (tm *TemplateManager) Save() error {
	data, err := json.MarshalIndent(tm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TemplatesFile, data, 0644)
}

// GetTemplate retrieves a template by name
func (tm *TemplateManager) GetTemplate(name string) *Template {
	for i, t := range tm.Templates {
		if t.Name == name {
			return &tm.Templates[i]
		}
	}
	return nil
}

// AddTemplate adds a new template
func (tm *TemplateManager) AddTemplate(t Template) error {
	// Check for duplicate name
	if tm.GetTemplate(t.Name) != nil {
		return fmt.Errorf("template '%s' already exists", t.Name)
	}

	// Extract variables from content
	t.Variables = extractVariables(t.Content)
	tm.Templates = append(tm.Templates, t)
	return tm.Save()
}

// UpdateTemplate updates an existing template
func (tm *TemplateManager) UpdateTemplate(name string, content string) error {
	for i, t := range tm.Templates {
		if t.Name == name {
			tm.Templates[i].Content = content
			tm.Templates[i].Variables = extractVariables(content)
			return tm.Save()
		}
	}
	return fmt.Errorf("template '%s' not found", name)
}

// DeleteTemplate removes a template
func (tm *TemplateManager) DeleteTemplate(name string) error {
	for i, t := range tm.Templates {
		if t.Name == name {
			tm.Templates = append(tm.Templates[:i], tm.Templates[i+1:]...)
			return tm.Save()
		}
	}
	return fmt.Errorf("template '%s' not found", name)
}

// ListTemplates returns all template names
func (tm *TemplateManager) ListTemplates() []string {
	names := make([]string, len(tm.Templates))
	for i, t := range tm.Templates {
		names[i] = t.Name
	}
	return names
}

// RenderTemplate fills in template variables
func (tm *TemplateManager) RenderTemplate(templateName string, vars map[string]string) (string, error) {
	t := tm.GetTemplate(templateName)
	if t == nil {
		return "", fmt.Errorf("template '%s' not found", templateName)
	}

	return RenderContent(t.Content, vars), nil
}

// RenderContent fills variables in any content string
func RenderContent(content string, vars map[string]string) string {
	result := content
	for key, value := range vars {
		// Support both {var} and {VAR} style
		result = strings.ReplaceAll(result, key, value)
		result = strings.ReplaceAll(result, strings.ToUpper(key), value)
	}
	return result
}

// extractVariables finds all {variable} patterns in content
func extractVariables(content string) []string {
	var vars []string
	seen := make(map[string]bool)

	inVar := false
	varStart := 0

	for i, ch := range content {
		if ch == '{' {
			inVar = true
			varStart = i
		} else if ch == '}' && inVar {
			varName := content[varStart : i+1]
			if !seen[varName] {
				vars = append(vars, varName)
				seen[varName] = true
			}
			inVar = false
		}
	}

	return vars
}

// ValidateVariables checks if all required variables are provided
func ValidateVariables(template *Template, vars map[string]string) []string {
	var missing []string
	for _, v := range template.Variables {
		if _, ok := vars[v]; !ok {
			missing = append(missing, v)
		}
	}
	return missing
}

// PrintTemplates displays all templates nicely
func (tm *TemplateManager) PrintTemplates() {
	fmt.Println("\n📝 Available Message Templates:")
	fmt.Println(strings.Repeat("-", 50))
	for _, t := range tm.Templates {
		fmt.Printf("\n📌 %s\n", t.Name)
		if t.Description != "" {
			fmt.Printf("   %s\n", t.Description)
		}
		fmt.Printf("   Variables: %v\n", t.Variables)
		fmt.Printf("   Preview: %.80s...\n", t.Content)
	}
	fmt.Println(strings.Repeat("-", 50))
}
