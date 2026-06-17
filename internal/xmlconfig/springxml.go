package xmlconfig

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// SpringBeans represents a Spring beans XML configuration file.
type SpringBeans struct {
	XMLName xml.Name       `xml:"beans"`
	Beans   []SpringBean   `xml:"bean"`
	MVC     *SpringMVC     `xml:"annotation-driven"`
	Scan    *SpringScan    `xml:"component-scan"`
	Imports []SpringImport `xml:"import"`
}

// SpringBean represents a <bean> element.
type SpringBean struct {
	ID       string          `xml:"id,attr"`
	Class    string          `xml:"class,attr"`
	Name     string          `xml:"name,attr"`
	Scope    string          `xml:"scope,attr"`
	Props    []SpringProperty `xml:"property"`
	Ref      string          `xml:"ref,attr"`
}

// SpringProperty represents a <property> element.
type SpringProperty struct {
	Name string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Ref  string `xml:"ref,attr"`
}

// SpringMVC represents a <mvc:annotation-driven> element.
type SpringMVC struct {
	XMLName xml.Name `xml:"annotation-driven"`
}

// SpringScan represents a <context:component-scan> element.
type SpringScan struct {
	XMLName     xml.Name `xml:"component-scan"`
	BasePackage string   `xml:"base-package,attr"`
}

// SpringImport represents an <import> element.
type SpringImport struct {
	Resource string `xml:"resource,attr"`
}

// SpringFactories represents a spring.factories file.
type SpringFactories struct {
	Factories map[string][]string // key → list of implementation classes
}

// ParseSpringBeans parses a Spring beans XML file.
func ParseSpringBeans(data []byte) (*SpringBeans, error) {
	var b SpringBeans
	if err := xml.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse spring beans: %w", err)
	}
	return &b, nil
}

// ParseSpringFactories parses a spring.factories file.
// Format: key = value1,value2,...
func ParseSpringFactories(data []byte) *SpringFactories {
	sf := &SpringFactories{
		Factories: make(map[string][]string),
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Handle line continuation (trailing \).
		fullLine := line
		for strings.HasSuffix(fullLine, "\\") {
			fullLine = fullLine[:len(fullLine)-1]
			// In practice continuation lines would need more state; basic support only.
		}

		idx := strings.Index(fullLine, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(fullLine[:idx])
		values := strings.TrimSpace(fullLine[idx+1:])
		parts := strings.Split(values, ",")
		var cleaned []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cleaned = append(cleaned, p)
			}
		}
		sf.Factories[key] = cleaned
	}
	return sf
}

// HasSpringBootAutoConfig checks if spring.factories contains Spring Boot auto-configuration.
func (sf *SpringFactories) HasSpringBootAutoConfig() bool {
	keys := []string{
		"org.springframework.boot.autoconfigure.EnableAutoConfiguration",
		"org.springframework.context.ApplicationContextInitializer",
	}
	for _, k := range keys {
		if _, ok := sf.Factories[k]; ok {
			return true
		}
	}
	return false
}

// IsSpringBootLauncher checks if a class is likely a Spring Boot launcher.
func IsSpringBootLauncher(className string) bool {
	bootLaunchers := []string{
		"org.springframework.boot.loader.JarLauncher",
		"org.springframework.boot.loader.WarLauncher",
		"org.springframework.boot.loader.PropertiesLauncher",
	}
	for _, bl := range bootLaunchers {
		if className == bl {
			return true
		}
	}
	return false
}
