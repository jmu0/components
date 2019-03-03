package components

import (
	"strings"
)

//Page struct for page data
type Page struct {
	Route      string
	Components []Part
}

//Render renders the components
func (p *Page) Render(components map[string]Component, path string) (string, error) {
	var html string
	var scripts []string
	for _, comp := range p.Components {
		cmphtml, err := comp.Render(components, path)
		if err != nil {
			return "", err
		}
		scripts = append(scripts, comp.ScriptTags(components, true)...) //TODO debug value
		html += cmphtml
	}
	if strings.Contains(html, "</body>") {
		html = strings.Replace(html, "</body>", strings.Join(scripts, "\n")+"\n</body>", 1)
	} else {
		html += strings.Join(scripts, "\n")
	}
	return html, nil
}
