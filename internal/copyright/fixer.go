package copyright

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YakDriver/copyplop/internal/config"
	"github.com/schollz/progressbar/v3"
)

type Fixer struct {
	config *config.Config
}

func NewFixer(cfg *config.Config) *Fixer {
	return &Fixer{config: cfg}
}

func (f *Fixer) Fix(path string) (*FixResult, error) {
	files, err := getTrackedFiles(path)
	if err != nil {
		return nil, err
	}

	// Filter files to process
	var filesToProcess []string
	for _, file := range files {
		if f.config.ShouldProcess(file) {
			filesToProcess = append(filesToProcess, file)
		}
	}

	if len(filesToProcess) == 0 {
		return &FixResult{}, nil
	}

	bar := progressbar.Default(int64(len(filesToProcess)), "Fixing files")
	result := &FixResult{}

	for _, file := range filesToProcess {
		if f.fixFile(file) {
			result.Fixed++
		}
		bar.Add(1)
	}

	return result, nil
}

func (f *Fixer) fixFile(file string) bool {
	content, err := os.ReadFile(file)
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || f.config.IsGenerated(lines) {
		return false
	}

	// Get extension, handling compound extensions like .html.markdown
	ext := filepath.Ext(file)
	for _, validExt := range f.config.Files.Extensions {
		if strings.HasSuffix(file, validExt) && len(validExt) > len(ext) {
			ext = validExt
			break
		}
	}

	copyrightHeader, err := f.config.GetCopyrightHeader(ext)
	if err != nil {
		return false
	}

	licenseHeader, err := f.config.GetLicenseHeader(ext)
	if err != nil {
		return false
	}

	var result []string
	startLine := 0
	fixed := false
	hasCopyright := false
	thirdPartyLines := []string{}

	// Handle shebang
	if hasShebang(lines) {
		result = append(result, lines[0])
		startLine = 1
	}

	// Handle frontmatter
	frontmatterEnd := getFrontmatterEnd(lines, f.config, file)
	if frontmatterEnd > startLine {
		// Add frontmatter to result
		result = append(result, lines[startLine:frontmatterEnd]...)
		startLine = frontmatterEnd
	}

	// Scan for existing copyrights and third-party copyrights
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		if f.config.ShouldReplace(line) || strings.Contains(line, "SPDX-License-Identifier") {
			hasCopyright = true
		} else if f.config.IsThirdPartyCopyright(line) {
			thirdPartyLines = append(thirdPartyLines, line)
		} else if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) {
			return false // Already correct
		}
	}

	// Handle third-party copyrights based on action
	switch f.config.ThirdParty.Action {
	case "above":
		// Add our copyright above third-party
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, thirdPartyLines...)
		result = append(result, "")
	case "below":
		// Add third-party first, then our copyright
		result = append(result, thirdPartyLines...)
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	case "replace":
		// Replace third-party with our copyright
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	default: // "leave" or unspecified
		// Just add our copyright, leave third-party as-is
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	}

	// Process remaining content, skipping old headers
	skipNext := false
	for i := startLine; i < len(lines); i++ {
		line := lines[i]

		// Skip lines that should be replaced or are third-party (unless action is "leave")
		if f.config.ShouldReplace(line) || strings.Contains(line, "SPDX-License-Identifier") {
			fixed = true
			skipNext = true
			continue
		}

		if f.config.IsThirdPartyCopyright(line) && f.config.ThirdParty.Action != "leave" {
			fixed = true
			skipNext = true
			continue
		}

		// Skip empty lines after headers
		if skipNext && strings.TrimSpace(line) == "" {
			skipNext = false
			continue
		}

		skipNext = false
		result = append(result, line)
	}

	// If no copyright was found, we're adding new headers
	if !hasCopyright && len(thirdPartyLines) == 0 {
		fixed = true
	} else if len(thirdPartyLines) > 0 {
		fixed = true // Always fix when third-party copyright is present
	}

	if fixed {
		newContent := strings.Join(result, "\n")
		os.WriteFile(file, []byte(newContent), 0644)
		fmt.Printf("Fixed: %s\n", file)
		return true
	}

	return false
}
