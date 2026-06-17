package archive

import (
	"bufio"
	"bytes"
	"strings"
)

// Manifest represents a parsed MANIFEST.MF file.
type Manifest struct {
	Main     map[string]string
	Sections map[string]map[string]string
}

// ParseManifest parses MANIFEST.MF content.
// Lines starting with SPACE are continuations of the previous line.
// Sections are separated by blank lines; named sections start with "Name: value".
func ParseManifest(data []byte) (*Manifest, error) {
	m := &Manifest{
		Main:     make(map[string]string),
		Sections: make(map[string]map[string]string),
	}

	var currentSection map[string]string = m.Main
	var inMainSection = true
	var currentKey string
	var currentValue strings.Builder

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line ends the current section; next "Name:" starts a new section.
			// Flush any pending key-value.
			if currentKey != "" {
				trimmed := strings.TrimSpace(currentValue.String())
				if trimmed != "" {
					currentSection[currentKey] = trimmed
				}
				currentKey = ""
				currentValue.Reset()
			}
			continue
		}

		if line[0] == ' ' {
			// Continuation line — append to previous value.
			currentValue.WriteString(line[1:])
			continue
		}

		// New key-value pair. Flush previous first.
		if currentKey != "" {
			trimmed := strings.TrimSpace(currentValue.String())
			if trimmed != "" {
				currentSection[currentKey] = trimmed
			}
		}

		colonIdx := strings.IndexByte(line, ':')
		if colonIdx < 0 {
			continue
		}
		key := line[:colonIdx]
		value := strings.TrimSpace(line[colonIdx+1:])

		// Check if this is a named section header.
		if key == "Name" && inMainSection {
			inMainSection = false
			sectionName := value
			m.Sections[sectionName] = make(map[string]string)
			currentSection = m.Sections[sectionName]
		}

		currentKey = key
		currentValue.Reset()
		currentValue.WriteString(value)
	}

	// Flush final key-value.
	if currentKey != "" {
		trimmed := strings.TrimSpace(currentValue.String())
		if trimmed != "" {
			currentSection[currentKey] = trimmed
		}
	}

	return m, scanner.Err()
}

// Get returns the value for a key in the main section.
func (m *Manifest) Get(key string) string {
	return m.Main[key]
}

// GetSection returns a value from a named section.
func (m *Manifest) GetSection(name, key string) string {
	if s, ok := m.Sections[name]; ok {
		return s[key]
	}
	return ""
}

// HasBootInf returns true if this manifest indicates a Spring Boot app
// (has Spring-Boot-Classes or Spring-Boot-Lib header).
func (m *Manifest) HasBootInf() bool {
	return m.Get("Spring-Boot-Classes") != "" || m.Get("Spring-Boot-Lib") != ""
}

// BootVersion returns the Spring Boot version from the manifest, if present.
func (m *Manifest) BootVersion() string {
	return m.Get("Spring-Boot-Version")
}

// MainClass returns the Main-Class from the manifest.
func (m *Manifest) MainClass() string {
	return m.Get("Main-Class")
}
