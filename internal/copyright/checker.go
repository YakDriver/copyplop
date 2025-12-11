// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package copyright

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/YakDriver/copyplop/internal/config"
	"github.com/schollz/progressbar/v3"
)

type Checker struct {
	config *config.Config
}

func NewChecker(cfg *config.Config) *Checker {
	return &Checker{config: cfg}
}

func (c *Checker) Check(path string) ([]Issue, error) {
	files, err := getTrackedFiles(path, c.config)
	if err != nil {
		return nil, err
	}

	// Filter files to process
	var filesToProcess []string
	for _, file := range files {
		if c.config.ShouldProcess(file) {
			filesToProcess = append(filesToProcess, file)
		}
	}

	if len(filesToProcess) == 0 {
		return nil, nil
	}

	bar := progressbar.Default(int64(len(filesToProcess)), "Checking files")
	var issues []Issue

	for _, file := range filesToProcess {
		if issue := c.checkFile(file); issue != nil {
			issues = append(issues, *issue)
		}
		_ = bar.Add(1)
	}

	return issues, nil
}

func (c *Checker) checkFile(file string) *Issue {
	content, err := os.ReadFile(file)
	if err != nil {
		return &Issue{File: file, Problem: "could not read file"}
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return &Issue{File: file, Problem: "empty file"}
	}

	if c.config.IsGenerated(lines) {
		return nil
	}

	ext := filepath.Ext(file)
	expectedHeader, err := c.config.GetCopyrightHeader(ext)
	if err != nil {
		return &Issue{File: file, Problem: "config error: " + err.Error()}
	}

	expectedLicense, err := c.config.GetLicenseHeader(ext)
	if err != nil {
		return &Issue{File: file, Problem: "config error: " + err.Error()}
	}

	startLine := 0
	if hasShebang(lines) {
		startLine = 1
	}

	frontmatterEnd := getFrontmatterEnd(lines, c.config, file)
	if frontmatterEnd > startLine {
		startLine = frontmatterEnd
	}

	if startLine >= len(lines) {
		return &Issue{File: file, Problem: "missing copyright header"}
	}

	// Determine scan limit
	maxScan := len(lines)
	if c.config.Detection.MaxScanLines > 0 {
		maxScan = min(startLine+c.config.Detection.MaxScanLines, len(lines))
	}

	// Check if copyright and license exist in header area
	foundCopyright := false
	foundLicense := false
	for i := startLine; i < maxScan; i++ {
		if strings.Contains(lines[i], strings.TrimSpace(expectedHeader[2:])) {
			foundCopyright = true
			if c.config.Detection.RequireAtTop && i != startLine {
				return &Issue{File: file, Problem: "copyright not at top of file"}
			}
		}
		if expectedLicense != "" && strings.Contains(lines[i], strings.TrimSpace(expectedLicense[2:])) {
			foundLicense = true
		}
	}

	if !foundCopyright {
		return &Issue{File: file, Problem: "missing or incorrect copyright header"}
	}

	if expectedLicense != "" && !foundLicense {
		return &Issue{File: file, Problem: "missing license header"}
	}

	return nil
}
