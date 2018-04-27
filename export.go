package menu

import (
	"os"
	"text/template"
)

// Export render config files
func Export(t *template.Template, ta TemplateArgs) error {
	return t.Execute(os.Stdout, ta)
}
