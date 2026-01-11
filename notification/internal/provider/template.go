package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aymerick/raymond"
)

// TemplateEngine handles email template rendering
type TemplateEngine struct {
	templatesDir string
	cache        map[string]*raymond.Template
	mu           sync.RWMutex
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(templatesDir string) (*TemplateEngine, error) {
	// Verify templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("templates directory does not exist: %s", templatesDir)
	}

	// Register custom helpers
	raymond.RegisterHelper("formatCurrency", func(amount float64, currency string) string {
		if currency == "" {
			currency = "USD"
		}
		return fmt.Sprintf("%.2f %s", amount, currency)
	})

	raymond.RegisterHelper("formatDate", func(date string) string {
		return date
	})

	raymond.RegisterHelper("eq", func(a, b interface{}) bool {
		return a == b
	})

	raymond.RegisterHelper("gt", func(a, b float64) bool {
		return a > b
	})

	return &TemplateEngine{
		templatesDir: templatesDir,
		cache:        make(map[string]*raymond.Template),
	}, nil
}

// RenderTemplate renders a template with the given data
func (t *TemplateEngine) RenderTemplate(templateName string, data interface{}) (string, error) {
	template, err := t.getTemplate(templateName)
	if err != nil {
		return "", err
	}

	result, err := template.Exec(data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return result, nil
}

// getTemplate retrieves a template from cache or loads it from disk
func (t *TemplateEngine) getTemplate(templateName string) (*raymond.Template, error) {
	// Check cache first
	t.mu.RLock()
	if tmpl, ok := t.cache[templateName]; ok {
		t.mu.RUnlock()
		return tmpl, nil
	}
	t.mu.RUnlock()

	// Load template from disk
	templatePath := filepath.Join(t.templatesDir, templateName+".hbs")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", templateName, err)
	}

	tmpl, err := raymond.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Cache the template
	t.mu.Lock()
	t.cache[templateName] = tmpl
	t.mu.Unlock()

	return tmpl, nil
}

// ClearCache clears the template cache
func (t *TemplateEngine) ClearCache() {
	t.mu.Lock()
	t.cache = make(map[string]*raymond.Template)
	t.mu.Unlock()
}

// PreloadTemplates preloads all templates from the templates directory
func (t *TemplateEngine) PreloadTemplates() error {
	entries, err := os.ReadDir(t.templatesDir)
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".hbs" {
			continue
		}

		templateName := name[:len(name)-4] // Remove .hbs extension
		if _, err := t.getTemplate(templateName); err != nil {
			return fmt.Errorf("failed to preload template %s: %w", templateName, err)
		}
	}

	return nil
}
