package components

import (
	"errors"
	"strings"
)

//Part stores component names for app struct
type Part struct {
	Name       string `json:"name" yaml:"name"`
	Template   string `json:"template" yaml:"template"`
	Components []Part `json:"components" yaml:"components"`
}

//Render renders part (recursive)
func (p *Part) Render(components map[string]Component, path string) (string, error) {
	var err error
	var html, itemhtml, cmpName string
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
		} else if len(data) > 1 {
			for i := range data {
				itemhtml, err = cmp.Render(p.Template, data[i])
				if err != nil {
					return "", err
				}
				html += itemhtml
			}
		}
		cmpName = strings.ToLower(p.Name)
		html = "<" + cmpName + " data-component='" + cmpName + "'>" + html + "</" + cmpName + ">"
		return html, err
	}
	return "", errors.New("Component not found for part: " + p.Name)
}
