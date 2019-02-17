package components

//Page struct for page data
type Page struct {
	Route      string
	Components []Part
}

//Render renders the components
func (p *Page) Render(components map[string]Component, path string) (string, error) {
	var html string
	for _, comp := range p.Components {
		cmphtml, err := comp.Render(components, path)
		if err != nil {
			return "", err
		}
		html += cmphtml
	}
	return html, nil
}
