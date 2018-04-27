package menu

import (
	"os"
	"text/template"
)

func Export(t *template.Template, ta TemplateArgs) error {
	return t.Execute(os.Stdout, ta)
}
