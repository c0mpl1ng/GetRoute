package xmlconfig

import (
	"encoding/xml"
)

// WebXML represents a parsed web.xml deployment descriptor.
type WebXML struct {
	XMLName      xml.Name       `xml:"web-app"`
	DisplayName  string         `xml:"display-name"`
	Servlets     []WebServlet   `xml:"servlet"`
	ServletMaps  []WebServletMapping `xml:"servlet-mapping"`
	Filters      []WebFilter    `xml:"filter"`
	FilterMaps   []WebFilterMapping  `xml:"filter-mapping"`
	Listeners    []WebListener  `xml:"listener"`
	ContextParams []WebContextParam  `xml:"context-param"`
	WelcomeFiles []string        `xml:"welcome-file-list>welcome-file"`
	ErrorPages   []WebErrorPage `xml:"error-page"`
}

// WebServlet represents a <servlet> element.
type WebServlet struct {
	Name        string `xml:"servlet-name"`
	Class       string `xml:"servlet-class"`
	LoadOnStartup string `xml:"load-on-startup"`
	DisplayName string `xml:"display-name"`
}

// WebServletMapping represents a <servlet-mapping> element.
type WebServletMapping struct {
	Name    string `xml:"servlet-name"`
	Pattern string `xml:"url-pattern"`
}

// WebFilter represents a <filter> element.
type WebFilter struct {
	Name  string `xml:"filter-name"`
	Class string `xml:"filter-class"`
}

// WebFilterMapping represents a <filter-mapping> element.
type WebFilterMapping struct {
	Name    string `xml:"filter-name"`
	Pattern string `xml:"url-pattern"`
}

// WebListener represents a <listener> element.
type WebListener struct {
	Class string `xml:"listener-class"`
}

// WebContextParam represents a <context-param> element.
type WebContextParam struct {
	Name  string `xml:"param-name"`
	Value string `xml:"param-value"`
}

// WebErrorPage represents an <error-page> element.
type WebErrorPage struct {
	Code     string `xml:"error-code"`
	Location string `xml:"location"`
}

// ParseWebXML parses web.xml content.
func ParseWebXML(data []byte) (*WebXML, error) {
	var w WebXML
	if err := xml.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// GetFilterClasses returns all filter class names from web.xml.
func (w *WebXML) GetFilterClasses() []string {
	var classes []string
	for _, f := range w.Filters {
		if f.Class != "" {
			classes = append(classes, f.Class)
		}
	}
	return classes
}

// HasFilterClass checks if web.xml contains a specific filter class.
func (w *WebXML) HasFilterClass(class string) bool {
	for _, f := range w.Filters {
		if f.Class == class {
			return true
		}
	}
	return false
}

// HasStrutsFilter checks if web.xml is configured for Struts2.
func (w *WebXML) HasStrutsFilter() bool {
	strutsFilters := []string{
		"org.apache.struts2.dispatcher.FilterDispatcher",
		"org.apache.struts2.dispatcher.ng.filter.StrutsPrepareAndExecuteFilter",
		"org.apache.struts.dispatcher.FilterDispatcher",
	}
	for _, sf := range strutsFilters {
		if w.HasFilterClass(sf) {
			return true
		}
	}
	return false
}
