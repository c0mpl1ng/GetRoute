package xmlconfig

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// StrutsXML represents a parsed struts.xml configuration.
type StrutsXML struct {
	XMLName   xml.Name         `xml:"struts"`
	Packages  []StrutsPackage  `xml:"package"`
	Constants []StrutsConstant `xml:"constant"`
	Includes  []StrutsInclude  `xml:"include"`
}

// StrutsPackage represents a <package> element in struts.xml/xwork.xml.
type StrutsPackage struct {
	Name      string         `xml:"name,attr"`
	Namespace string         `xml:"namespace,attr"`
	Extends   string         `xml:"extends,attr"`
	Abstract  string         `xml:"abstract,attr"`
	Actions   []StrutsAction `xml:"action"`
}

// StrutsAction represents an <action> element.
type StrutsAction struct {
	Name    string         `xml:"name,attr"`
	Class   string         `xml:"class,attr"`
	Method  string         `xml:"method,attr"`
	Results []StrutsResult `xml:"result"`
}

// StrutsResult represents a <result> element.
type StrutsResult struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

// StrutsConstant represents a <constant> element.
type StrutsConstant struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// StrutsInclude represents an <include> element.
type StrutsInclude struct {
	File string `xml:"file,attr"`
}

// XWorkXML represents a parsed xwork.xml (WebWork) configuration.
type XWorkXML struct {
	XMLName  xml.Name        `xml:"xwork"`
	Packages []StrutsPackage `xml:"package"`
	Includes []StrutsInclude `xml:"include"`
}

// ParseStrutsXML parses struts.xml content.
func ParseStrutsXML(data []byte) (*StrutsXML, error) {
	data = stripDocType(data)
	var s StrutsXML
	if err := xml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse struts.xml: %w", err)
	}
	return &s, nil
}

// ParseXWorkXML parses xwork.xml content.
func ParseXWorkXML(data []byte) (*XWorkXML, error) {
	data = stripDocType(data)
	var x XWorkXML
	if err := xml.Unmarshal(data, &x); err != nil {
		return nil, fmt.Errorf("parse xwork.xml: %w", err)
	}
	return &x, nil
}

// ParseXWorkOrStrutsXML tries to parse as XWork first, then Struts (both use <xwork> or <struts> root).
func ParseXWorkOrStrutsXML(data []byte) (*XWorkXML, error) {
	data = stripDocType(data)

	// Try as <xwork> first (WebWork).
	var xw XWorkXML
	if err := xml.Unmarshal(data, &xw); err == nil && len(xw.Packages) > 0 {
		return &xw, nil
	}

	// Try as <struts> (Struts2-style xwork within struts namespace).
	var sx StrutsXML
	if err := xml.Unmarshal(data, &sx); err == nil && len(sx.Packages) > 0 {
		return &XWorkXML{
			Packages: sx.Packages,
			Includes: sx.Includes,
		}, nil
	}

	// Last attempt: just try xwork again and return any error.
	var xw2 XWorkXML
	err := xml.Unmarshal(data, &xw2)
	return &xw2, err
}

// GetAllActions returns all actions across all packages.
func (s *StrutsXML) GetAllActions() []ResolvedAction {
	return resolveActions(s.Packages)
}

// GetAllActions returns all actions from xwork.xml.
func (x *XWorkXML) GetAllActions() []ResolvedAction {
	return resolveActions(x.Packages)
}

// ResolvedAction represents an action with its resolved namespace.
type ResolvedAction struct {
	PackageName     string
	Namespace       string
	ActionName      string
	ActionClass     string
	ActionMethod    string
	DefaultMethod   string
	ActionExtension string // Default ".action"
}

func resolveActions(packages []StrutsPackage) []ResolvedAction {
	var actions []ResolvedAction
	for _, pkg := range packages {
		ns := normalizeStrutsNamespace(pkg.Namespace)
		for _, action := range pkg.Actions {
			method := action.Method
			if method == "" {
				method = "execute"
			}
			actions = append(actions, ResolvedAction{
				PackageName:     pkg.Name,
				Namespace:       ns,
				ActionName:      action.Name,
				ActionClass:     action.Class,
				ActionMethod:    method,
				DefaultMethod:   "execute",
				ActionExtension: ".action",
			})
		}
	}
	return actions
}

// URL returns the full URL path for a resolved action.
func (a *ResolvedAction) URL() string {
	path := a.Namespace
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	actionName := a.ActionName

	// Handle wildcard actions like "user_*".
	if strings.Contains(actionName, "*") {
		actionName = strings.ReplaceAll(actionName, "*", "{wildcard}")
	}

	// Append action name.
	if strings.HasSuffix(path, "/") {
		path += actionName
	} else {
		path += "/" + actionName
	}

	// Add action extension (.action or .do).
	path += a.ActionExtension

	// Handle dynamic method invocation patterns: actionName!method.
	if a.ActionMethod != "" && a.ActionMethod != a.DefaultMethod {
		path += "!" + a.ActionMethod
	}

	return normalizePath(path)
}

func normalizeStrutsNamespace(ns string) string {
	if ns == "" {
		return "/"
	}
	if ns == "/" {
		return "/"
	}
	if !strings.HasPrefix(ns, "/") {
		ns = "/" + ns
	}
	return ns
}

func normalizePath(path string) string {
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

// StrutsVersion attempts to determine the Struts version from constants.
func (s *StrutsXML) StrutsVersion() string {
	for _, c := range s.Constants {
		if c.Name == "struts.devMode" || c.Name == "struts.configuration.xml.reload" {
			return "2.x"
		}
	}
	return ""
}

// IsXWorkFile checks if a filename is an xwork configuration file.
func IsXWorkFile(name string) bool {
	lower := strings.ToLower(name)
	base := lower
	// Get the base filename without directory.
	if idx := strings.LastIndex(lower, "/"); idx >= 0 {
		base = lower[idx+1:]
	}
	return strings.HasPrefix(base, "xwork") && strings.HasSuffix(base, ".xml")
}

// IsStrutsFile checks if a filename is a struts configuration file.
func IsStrutsFile(name string) bool {
	lower := strings.ToLower(name)
	base := lower
	if idx := strings.LastIndex(lower, "/"); idx >= 0 {
		base = lower[idx+1:]
	}
	return strings.HasPrefix(base, "struts") && strings.HasSuffix(base, ".xml")
}

// stripDocType removes DOCTYPE declarations that Go's XML parser cannot handle.
func stripDocType(data []byte) []byte {
	// Remove <!DOCTYPE ...> declaration.
	if idx := bytes.Index(data, []byte("<!DOCTYPE")); idx >= 0 {
		end := bytes.IndexByte(data[idx:], '>')
		if end >= 0 {
			data = append(data[:idx], data[idx+end+1:]...)
		}
	}
	return data
}
