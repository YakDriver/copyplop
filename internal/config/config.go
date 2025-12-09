// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
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

type Files struct {
	Extensions       []string          `yaml:"extensions" mapstructure:"extensions"`
	IgnorePatterns   []string          `yaml:"ignore_patterns" mapstructure:"ignore_patterns"`
	IncludePaths     []string          `yaml:"include_paths" mapstructure:"include_paths"`
	ExcludePaths     []string          `yaml:"exclude_paths" mapstructure:"exclude_paths"`
	CommentStyles    map[string]string `yaml:"comment_styles" mapstructure:"comment_styles"`
	BelowFrontmatter []string          `yaml:"below_frontmatter" mapstructure:"below_frontmatter"`
	GitTracked       bool              `yaml:"git_tracked" mapstructure:"git_tracked"`
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

	return prefix + " " + buf.String(), nil
}

func (c *Config) ShouldProcess(file string) bool {
	// Check extension first
	hasValidExt := false
	for _, validExt := range c.Files.Extensions {
		if strings.HasSuffix(file, validExt) {
			hasValidExt = true
			break
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

// matchesPath checks if a file path matches a pattern, supporting directory patterns
func matchesPath(pattern, path string) bool {
	// Direct match
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	
	// Check if pattern matches any parent directory path
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		if matched, _ := filepath.Match(pattern, dir); matched {
			return true
		}
		// Also check if the pattern matches the directory plus wildcard
		if matched, _ := filepath.Match(pattern, dir+"/*"); matched {
			return true
		}
		dir = filepath.Dir(dir)
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
