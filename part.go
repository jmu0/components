package components

import (
	"errors"
)

//Part stores component names for app struct
type Part struct {
	Name       string
	Template   string
	Components []Part
}

//Render renders part (recursive)
func (p *Part) Render(components map[string]Component, path string) (string, error) {
	var err error

	if cmp, ok := components[p.Name]; ok {
		var partData = make(map[string]interface{})
		for _, prt := range p.Components {
			partData[prt.Name], err = prt.Render(components, path)
			if err != nil {
				return "", err
			}
		}
		if p.Template == "" {
			for tmpl := range cmp.TemplateManager.GetTemplates() {
				p.Template = tmpl
				break
			}
		}
		data, err := cmp.GetData(path)
		if err != nil {
			data = partData
		} else {
			for k, v := range partData {
				data[k] = v
			}
		}
		return cmp.Render(p.Template, data)
	}
	return "", errors.New("Component not found for part: " + p.Name)
}
