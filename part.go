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
	var html, itemhtml string
	var data []map[string]interface{}
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
		data, err = cmp.GetData(path)
		if err != nil || len(data) == 0 {
			data = make([]map[string]interface{}, 0)
			data = append(data, partData)
		} else {
			for i := range data {
				for k, v := range partData {
					data[i][k] = v
				}
			}
		}
		if len(data) <= 1 {
			d := make(map[string]interface{})
			if len(data) == 1 {
				d = data[0]
			}
			html, err = cmp.Render(p.Template, d)
			return html, err
		} else if len(data) > 1 {
			for i := range data {
				itemhtml, err = cmp.Render(p.Template, data[i])
				if err != nil {
					return "", err
				}
				html += itemhtml
			}
			return html, nil
		}
	}
	return "", errors.New("Component not found for part: " + p.Name)
}
