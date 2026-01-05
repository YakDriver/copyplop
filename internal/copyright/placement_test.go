package copyright

import (
	"testing"

	"github.com/YakDriver/copyplop/internal/config"
)

func TestPlacementExceptions(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		config   *config.Config
		expected int // expected startLine after processing exceptions
	}{
		{
			name:  "XML declaration enabled",
			lines: []string{"<?xml version=\"1.0\"?>", "<!-- content -->"},
			config: &config.Config{
				Files: config.Files{
					PlacementExceptions: config.PlacementExceptions{
						XMLDeclaration: true,
					},
				},
			},
			expected: 1,
		},
		{
			name:  "XML declaration disabled",
			lines: []string{"<?xml version=\"1.0\"?>", "<!-- content -->"},
			config: &config.Config{
				Files: config.Files{
					PlacementExceptions: config.PlacementExceptions{
						XMLDeclaration: false,
					},
				},
			},
			expected: 0,
		},
		{
			name:  "Markdown heading enabled",
			lines: []string{"# Title", "content"},
			config: &config.Config{
				Files: config.Files{
					PlacementExceptions: config.PlacementExceptions{
						MarkdownHeading: true,
					},
				},
			},
			expected: 1,
		},
		{
			name:  "Combined exceptions",
			lines: []string{"#!/bin/bash", "<?xml version=\"1.0\"?>", "# Title", "content"},
			config: &config.Config{
				Files: config.Files{
					PlacementExceptions: config.PlacementExceptions{
						XMLDeclaration:  true,
						MarkdownHeading: true,
					},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startLine := 0

			// Simulate the exception processing logic
			if hasShebang(tt.lines) {
				startLine = 1
			}

			if startLine < len(tt.lines) && tt.config.Files.PlacementExceptions.XMLDeclaration && hasXMLDeclaration(tt.lines[startLine:]) {
				startLine++
			}

			if startLine < len(tt.lines) && tt.config.Files.PlacementExceptions.MarkdownHeading && hasMarkdownHeading(tt.lines[startLine:]) {
				startLine++
			}

			if startLine != tt.expected {
				t.Errorf("expected startLine %d, got %d", tt.expected, startLine)
			}
		})
	}
}

func TestDetectionFunctions(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		function func([]string) bool
		expected bool
	}{
		{
			name:     "XML declaration detected",
			lines:    []string{"<?xml version=\"1.0\"?>"},
			function: hasXMLDeclaration,
			expected: true,
		},
		{
			name:     "XML declaration with whitespace",
			lines:    []string{"  <?xml version=\"1.0\"?>"},
			function: hasXMLDeclaration,
			expected: true,
		},
		{
			name:     "Not XML declaration",
			lines:    []string{"<xml>"},
			function: hasXMLDeclaration,
			expected: false,
		},
		{
			name:     "Markdown heading detected",
			lines:    []string{"# Title"},
			function: hasMarkdownHeading,
			expected: true,
		},
		{
			name:     "Markdown heading with whitespace",
			lines:    []string{"  # Title"},
			function: hasMarkdownHeading,
			expected: true,
		},
		{
			name:     "Not markdown heading",
			lines:    []string{"## Subtitle"},
			function: hasMarkdownHeading,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.lines)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
