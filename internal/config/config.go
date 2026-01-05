// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"

	"github.com/bmatcuk/doublestar/v4"
)

type Config struct {
	Copyright  Copyright  `yaml:"copyright"`
	License    License    `yaml:"license"`
	Files      Files      `yaml:"files"`
	Detection  Detection  `yaml:"detection"`
	ThirdParty ThirdParty `yaml:"third_party"`
}

type Copyright struct {
	Holder      string `yaml:"holder" mapstructure:"holder"`
	StartYear   int    `yaml:"start_year" mapstructure:"start_year"`
	CurrentYear int    `yaml:"current_year" mapstructure:"current_year"`
	Format      string `yaml:"format" mapstructure:"format"`
}

type License struct {
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`
	Identifier string `yaml:"identifier" mapstructure:"identifier"`
	Format     string `yaml:"format" mapstructure:"format"`
}

type SmartExtensionIndicators struct {
	Extension string   `yaml:"extension" mapstructure:"extension"`
	Patterns  []string `yaml:"patterns" mapstructure:"patterns"`
	Filenames []string `yaml:"filenames" mapstructure:"filenames"`
}

type PlacementExceptions struct {
	XMLDeclaration  bool     `yaml:"xml_declaration" mapstructure:"xml_declaration"`
	MarkdownHeading bool     `yaml:"markdown_heading" mapstructure:"markdown_heading"`
	Frontmatter     []string `yaml:"frontmatter" mapstructure:"frontmatter"`
}

type Files struct {
	Extensions               []string                   `yaml:"extensions" mapstructure:"extensions"`
	SmartExtensions          []string                   `yaml:"smart_extensions" mapstructure:"smart_extensions"`
	SmartExtensionIndicators []SmartExtensionIndicators `yaml:"smart_extension_indicators" mapstructure:"smart_extension_indicators"`
	IgnorePatterns           []string                   `yaml:"ignore_patterns" mapstructure:"ignore_patterns"`
	IncludePaths             []string                   `yaml:"include_paths" mapstructure:"include_paths"`
	ExcludePaths             []string                   `yaml:"exclude_paths" mapstructure:"exclude_paths"`
	CommentStyles            map[string]string          `yaml:"comment_styles" mapstructure:"comment_styles"`
	BelowFrontmatter         []string                   `yaml:"below_frontmatter" mapstructure:"below_frontmatter"`
	PlacementExceptions      PlacementExceptions        `yaml:"placement_exceptions" mapstructure:"placement_exceptions"`
	GitTracked               bool                       `yaml:"git_tracked" mapstructure:"git_tracked"`
}

type Detection struct {
	SkipGenerated     bool     `yaml:"skip_generated" mapstructure:"skip_generated"`
	GeneratedPatterns []string `yaml:"generated_patterns" mapstructure:"generated_patterns"`
	ReplacePatterns   []string `yaml:"replace_patterns" mapstructure:"replace_patterns"`
	MaxScanLines      int      `yaml:"max_scan_lines" mapstructure:"max_scan_lines"`
	RequireAtTop      bool     `yaml:"require_at_top" mapstructure:"require_at_top"`
}

type ThirdParty struct {
	Action   string   `yaml:"action" mapstructure:"action"`
	Patterns []string `yaml:"patterns" mapstructure:"patterns"`
}

func (c *Config) GetCopyrightHeader(ext string) (string, error) {
	tmpl, err := template.New("copyright").Parse(c.Copyright.Format)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, c.Copyright)
	if err != nil {
		return "", err
	}

	// Remove the dot from extension for lookup
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	prefix := c.Files.CommentStyles[extKey]
	if prefix == "" {
		// Fallback to hardcoded values if not found in config
		switch ext {
		case ".go":
			prefix = "//"
		case ".sh", ".py", ".hcl", ".tf", ".yml", ".yaml":
			prefix = "#"
		case ".md", ".html.markdown":
			prefix = "<!--"
		default:
			prefix = "//"
		}
	}

	// Special case: HTML/markdown comments need closing -->
	if prefix == "<!--" {
		return prefix + " " + buf.String() + " -->", nil
	}

	// Special case: JS/CSS block comments
	if prefix == "/**" {
		return " * " + buf.String(), nil
	}

	// Special case: YAML files need quotes around comments containing colons
	if ext == ".yml" || ext == ".yaml" {
		content := buf.String()
		if strings.Contains(content, ":") {
			return prefix + " \"" + content + "\"", nil
		}
	}

	return prefix + " " + buf.String(), nil
}

func (c *Config) GetLicenseHeader(ext string) (string, error) {
	if !c.License.Enabled {
		return "", nil
	}

	tmpl, err := template.New("license").Parse(c.License.Format)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, c.License)
	if err != nil {
		return "", err
	}

	// Remove the dot from extension for lookup
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	prefix := c.Files.CommentStyles[extKey]
	if prefix == "" {
		// Fallback to hardcoded values if not found in config
		switch ext {
		case ".go":
			prefix = "//"
		case ".sh", ".py", ".hcl", ".tf", ".yml", ".yaml":
			prefix = "#"
		case ".md", ".html.markdown":
			prefix = "<!--"
		default:
			prefix = "//"
		}
	}

	// Special case: HTML/markdown comments need closing -->
	if prefix == "<!--" {
		return prefix + " " + buf.String() + " -->", nil
	}

	// Special case: JS/CSS block comments
	if prefix == "/**" {
		return " * " + buf.String(), nil
	}

	// Special case: YAML files need quotes around comments containing colons
	if ext == ".yml" || ext == ".yaml" {
		content := buf.String()
		if strings.Contains(content, ":") {
			return prefix + " \"" + content + "\"", nil
		}
	}

	return prefix + " " + buf.String(), nil
}

func (c *Config) ShouldProcess(file string) bool {
	// Check extension first
	hasValidExt := false

	// Check regular extensions
	for _, validExt := range c.Files.Extensions {
		if strings.HasSuffix(file, validExt) {
			hasValidExt = true
			break
		}
	}

	// Check smart extensions
	if !hasValidExt {
		for _, smartExt := range c.Files.SmartExtensions {
			if strings.HasSuffix(file, smartExt) {
				hasValidExt = true
				break
			}
		}
	}

	if !hasValidExt {
		return false
	}

	// Apply path filtering logic
	return c.shouldProcessPath(file)
}

// shouldProcessPath implements the include/exclude path logic:
// - No includes + no excludes = process everything
// - Has includes = only process files matching includes
// - Has excludes = process everything except excludes
// - Has both = process files that match includes AND don't match excludes
func (c *Config) shouldProcessPath(file string) bool {
	hasIncludes := len(c.Files.IncludePaths) > 0
	hasExcludes := len(c.Files.ExcludePaths) > 0

	// No path filters = process everything
	if !hasIncludes && !hasExcludes {
		return true
	}

	// Check excludes first (if any)
	if hasExcludes {
		for _, pattern := range c.Files.ExcludePaths {
			if matchesPath(pattern, file) {
				return false
			}
		}
	}

	// Check includes (if any)
	if hasIncludes {
		for _, pattern := range c.Files.IncludePaths {
			if matchesPath(pattern, file) {
				return true
			}
		}
		return false // Has includes but file didn't match any
	}

	// Has excludes but no includes, and file didn't match excludes
	return true
}

// matchesPath checks if a file path matches a pattern, supporting doublestar glob patterns
func matchesPath(pattern, path string) bool {
	// Try exact match first
	if matched, _ := doublestar.Match(pattern, path); matched {
		return true
	}

	// Handle directory patterns by converting /* to /** for subdirectory matching
	if before, ok := strings.CutSuffix(pattern, "/*"); ok {
		// Convert "internal/service/*" to "internal/service/**" to match subdirectories
		dirPattern := before + "/**"
		if matched, _ := doublestar.Match(dirPattern, path); matched {
			return true
		}
	} else if !strings.HasSuffix(pattern, "/**") {
		// For patterns not ending with /** or /*, try adding /** to match files in subdirectories
		if matched, _ := doublestar.Match(pattern+"/**", path); matched {
			return true
		}
	}

	return false
}

func (c *Config) IsGenerated(lines []string) bool {
	if !c.Detection.SkipGenerated || len(lines) == 0 {
		return false
	}

	for _, pattern := range c.Detection.GeneratedPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(lines[0]) || (len(lines) > 1 && re.MatchString(lines[1])) {
			return true
		}
	}
	return false
}

func (c *Config) ShouldReplace(line string) bool {
	for _, pattern := range c.Detection.ReplacePatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func (c *Config) IsThirdPartyCopyright(line string) bool {
	// First check if it matches replacement patterns - if so, NOT third-party
	if c.ShouldReplace(line) {
		return false
	}

	// Then check third-party patterns
	for _, pattern := range c.ThirdParty.Patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// IsOwnCopyrightLine checks if a line matches our own copyright format (for self-updating)
func (c *Config) IsOwnCopyrightLine(line, ext string) bool {
	// Get comment prefix for this extension
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	prefix := c.Files.CommentStyles[extKey]
	if prefix == "" {
		// Fallback to hardcoded values if not found in config
		switch ext {
		case ".go":
			prefix = "//"
		case ".sh", ".py", ".hcl", ".tf", ".yml", ".yaml":
			prefix = "#"
		case ".md", ".html.markdown":
			prefix = "<!--"
		default:
			prefix = "//"
		}
	}

	var content string

	// Handle block comment style - don't trim spaces first
	if prefix == "/**" {
		if after, ok := strings.CutPrefix(line, " * "); ok {
			content = strings.TrimSpace(after)
		} else {
			return false
		}
	} else {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, prefix); ok {
			// Extract content after comment prefix
			content = strings.TrimSpace(after)

			// Handle HTML-style comments
			if prefix == "<!--" {
				content = strings.TrimSuffix(content, "-->")
				content = strings.TrimSpace(content)
			}
		} else {
			return false
		}
	}

	// Check if it matches our copyright pattern: "Copyright <holder> <years>"
	copyrightPattern := `^Copyright\s+` + regexp.QuoteMeta(c.Copyright.Holder) + `\s+\d{4}(,\s*\d{4})?$`
	matched, _ := regexp.MatchString(copyrightPattern, content)
	return matched
}

// DetectSmartExtensionType analyzes content to determine the actual file type for smart extensions
func (c *Config) DetectSmartExtensionType(content []byte, filename string) string {
	// Skip binary files - check for null bytes in first 512 bytes
	checkLen := min(len(content), 512)
	for i := range checkLen {
		if content[i] == 0 {
			return ""
		}
	}

	contentStr := string(content)

	// Use scoring system if indicators are configured
	if len(c.Files.SmartExtensionIndicators) > 0 {
		return c.detectByScoring(contentStr, filename)
	}

	// Fallback to original detection logic
	return c.detectByPatterns(contentStr, filename)
}

// detectByScoring uses configurable indicators to score each extension type
func (c *Config) detectByScoring(content, filename string) string {
	scores := make(map[string]int)

	for _, indicator := range c.Files.SmartExtensionIndicators {
		score := 0

		// Check content patterns
		for _, pattern := range indicator.Patterns {
			if strings.Contains(content, pattern) {
				score++
			}
		}

		// Check filename patterns
		for _, filenamePattern := range indicator.Filenames {
			if strings.Contains(filename, filenamePattern) {
				score++
			}
		}

		scores[indicator.Extension] = score
	}

	// Find extension with highest score
	maxScore := 0
	bestExt := ".go" // default
	for ext, score := range scores {
		if score > maxScore {
			maxScore = score
			bestExt = ext
		}
	}

	return bestExt
}

// detectByPatterns uses the original hardcoded detection logic
func (c *Config) detectByPatterns(content, filename string) string {
	// Check for Go code patterns
	if strings.Contains(content, "package ") ||
		strings.Contains(content, "func ") ||
		strings.Contains(content, "import (") ||
		strings.Contains(content, "type ") && strings.Contains(content, "struct") {
		return ".go"
	}

	// Check for HCL/Terraform patterns (before Markdown to avoid # comment confusion)
	if strings.Contains(content, "resource \"") ||
		strings.Contains(content, "data \"") ||
		strings.Contains(content, "variable \"") ||
		strings.Contains(content, "output \"") ||
		strings.Contains(content, "provider \"") ||
		strings.Contains(content, "terraform {") ||
		strings.Contains(filename, ".tf.") || // Files like test.tf.gtpl
		strings.Contains(filename, "terraform") {
		return ".tf"
	}

	// Check for Markdown patterns (more specific to avoid HCL # comments)
	if strings.Contains(content, "## ") ||
		strings.Contains(content, "### ") ||
		strings.Contains(content, "```") ||
		strings.Contains(content, "[") && strings.Contains(content, "](") ||
		(strings.Contains(content, "# ") && (strings.Contains(content, "layout:") || strings.Contains(content, "subcategory:"))) {
		return ".md"
	}

	// Check for YAML patterns
	if strings.Contains(content, "---") ||
		(strings.Contains(content, ":") && strings.Contains(content, "\n")) {
		return ".yml"
	}

	// Default fallback - could be based on filename patterns or directory
	if strings.Contains(filename, "markdown") || strings.Contains(filename, "md") {
		return ".md"
	}

	// Default to Go for unknown templates in terraform-provider-aws
	return ".go"
}
