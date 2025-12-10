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

type Fixer struct {
	config *config.Config
}

func NewFixer(cfg *config.Config) *Fixer {
	return &Fixer{config: cfg}
}

func (f *Fixer) Fix(path string) (*FixResult, error) {
	files, err := getTrackedFiles(path, f.config)
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
	
	// Check for smart extensions and detect actual content type
	isSmartExt := false
	for _, smartExt := range f.config.Files.SmartExtensions {
		if strings.HasSuffix(file, smartExt) && len(smartExt) > len(ext) {
			ext = smartExt
			isSmartExt = true
			break
		}
	}
	
	// For smart extensions, detect the actual file type from content
	if isSmartExt {
		detectedExt := f.config.DetectSmartExtensionType(content, file)
		if detectedExt == "" {
			// Binary file detected - skip processing
			return false
		}
		ext = detectedExt
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

	// Determine scan limit for header area
	maxScan := len(lines)
	if f.config.Detection.MaxScanLines > 0 {
		maxScan = min(startLine+f.config.Detection.MaxScanLines, len(lines))
	}

	// Scan for existing copyrights and third-party copyrights in header area only
	hasCorrectCopyright := false
	hasCorrectLicense := false
	for i := startLine; i < maxScan; i++ {
		line := lines[i]
		if f.config.ShouldReplace(line) {
			hasCopyright = true
		} else if f.config.IsThirdPartyCopyright(line) {
			thirdPartyLines = append(thirdPartyLines, line)
		} else if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) {
			hasCorrectCopyright = true
		} else if licenseHeader != "" && strings.TrimSpace(line) == strings.TrimSpace(licenseHeader) {
			hasCorrectLicense = true
		}
	}

	// If both copyright and license (if enabled) are already correct, nothing to do
	if hasCorrectCopyright && (licenseHeader == "" || hasCorrectLicense) {
		return false
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

	// Process remaining content, only removing copyrights from header area
	skipNext := false
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		inHeaderArea := i < maxScan

		// Only skip/remove copyright lines if in header area
		if inHeaderArea {
			// Remove old copyright/license lines if we're adding new ones
			if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) ||
				(licenseHeader != "" && strings.TrimSpace(line) == strings.TrimSpace(licenseHeader)) {
				skipNext = true
				continue
			}

			if f.config.ShouldReplace(line) {
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
		return true
	}

	return false
}

// ProcessContent applies the same header normalization logic as fixFile but on in-memory content
// This is primarily for testing the core logic without file I/O
func (f *Fixer) ProcessContent(content []byte, ext string) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || f.config.IsGenerated(lines) {
		return content, nil
	}

	copyrightHeader, err := f.config.GetCopyrightHeader(ext)
	if err != nil {
		return nil, err
	}

	licenseHeader, err := f.config.GetLicenseHeader(ext)
	if err != nil {
		return nil, err
	}

	var result []string
	startLine := 0
	thirdPartyLines := []string{}

	// Handle shebang (same as fixFile)
	if hasShebang(lines) {
		result = append(result, lines[0])
		startLine = 1
	}

	// Determine scan limit (same as fixFile)
	maxScan := len(lines)
	if f.config.Detection.MaxScanLines > 0 {
		maxScan = min(startLine+f.config.Detection.MaxScanLines, len(lines))
	}

	// Scan for third-party copyrights (same as fixFile)
	for i := startLine; i < maxScan; i++ {
		line := lines[i]
		if f.config.IsThirdPartyCopyright(line) {
			thirdPartyLines = append(thirdPartyLines, line)
		}
	}

	// Handle third-party copyrights (same as fixFile)
	switch f.config.ThirdParty.Action {
	case "above":
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, thirdPartyLines...)
		result = append(result, "")
	case "below":
		result = append(result, thirdPartyLines...)
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	case "replace":
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	default: // "leave"
		result = append(result, copyrightHeader)
		if licenseHeader != "" {
			result = append(result, licenseHeader)
		}
		result = append(result, "")
	}

	// Process remaining content (same logic as fixFile)
	skipNext := false
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		inHeaderArea := i < maxScan

		if inHeaderArea {
			if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) ||
				(licenseHeader != "" && strings.TrimSpace(line) == strings.TrimSpace(licenseHeader)) {
				skipNext = true
				continue
			}

			if f.config.ShouldReplace(line) {
				skipNext = true
				continue
			}

			if f.config.IsThirdPartyCopyright(line) && f.config.ThirdParty.Action != "leave" {
				skipNext = true
				continue
			}

			// Only skip blank lines immediately following a removed header line
			if skipNext && strings.TrimSpace(line) == "" {
				skipNext = false
				continue
			}
		}

		skipNext = false
		result = append(result, line)
	}

	output := strings.Join(result, "\n")

	// Preserve original trailing newline behavior
	if strings.HasSuffix(string(content), "\n") && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return []byte(output), nil
}
