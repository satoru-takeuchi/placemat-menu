package menu

import (
	"io"
	"text/template"
)

// Export render config files
func Export(t *template.Template, args interface{}, wr io.Writer) error {
	return t.Execute(wr, args)
}
