package components

import "github.com/jmu0/dbAPI/db"

//Page struct for page data
type Page struct {
	Route      string `json:"route" yaml:"route"`
	Auth       bool   `json:"auth" yaml:"auth"`
	Components []Part `json:"components" yaml:"components"`
}

//Render renders the components
func (p *Page) Render(path, locale string, components map[string]Component, conn db.Conn) (string, error) {
	var html string
	for _, comp := range p.Components {
		cmphtml, err := comp.Render(path, locale, components, conn)
		if err != nil {
			return "", err
		}
		html += cmphtml
	}
	return html, nil
}
