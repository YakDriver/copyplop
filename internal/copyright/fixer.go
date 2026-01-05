// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package copyright

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/YakDriver/copyplop/internal/config"
	"github.com/schollz/progressbar/v3"
)

// isBlockCommentStyle returns true if the comment style requires wrapping
func isBlockCommentStyle(cfg *config.Config, ext string) bool {
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	prefix := cfg.Files.CommentStyles[extKey]
	return prefix == "/**"
}

type Fixer struct {
	config *config.Config
}

func NewFixer(cfg *config.Config) *Fixer {
	return &Fixer{config: cfg}
}

// isSPDXHeaderLine detects SPDX-License-Identifier lines that are in comment format
func isSPDXHeaderLine(line, commentPrefix string) bool {
	var content string

	// Handle block comment style - don't trim spaces first
	if commentPrefix == "/**" {
		if after, ok := strings.CutPrefix(line, " * "); ok {
			content = strings.TrimSpace(after)
		} else {
			return false
		}
	} else {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, commentPrefix); ok {
			// Remove comment prefix and check for SPDX pattern
			content = strings.TrimSpace(after)
		} else {
			return false
		}
	}

	spdxPattern := `SPDX-License-Identifier:\s*"?[^"]*"?`
	matched, _ := regexp.MatchString(spdxPattern, content)
	return matched
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
		_ = bar.Add(1)
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
		if strings.HasSuffix(file, smartExt) && len(smartExt) >= len(ext) {
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

	// Handle shebang (always)
	if hasShebang(lines) {
		result = append(result, lines[0])
		startLine = 1
	}

	// Handle XML declaration
	if startLine < len(lines) && f.config.Files.PlacementExceptions.XMLDeclaration && hasXMLDeclaration(lines[startLine:]) {
		result = append(result, lines[startLine])
		startLine++
	}

	// Handle frontmatter - use detected extension for smart extensions
	frontmatterFile := file
	if isSmartExt {
		// For smart extensions, check frontmatter based on detected type
		// Use a filename that will match the BelowFrontmatter config
		frontmatterFile = "dummy" + ext
	}
	frontmatterEnd := getFrontmatterEndNew(lines, f.config, frontmatterFile)
	if frontmatterEnd > startLine {
		// Add frontmatter to result
		result = append(result, lines[startLine:frontmatterEnd]...)
		startLine = frontmatterEnd
	}

	// Handle markdown heading - only for markdown files
	isMarkdown := strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown")
	if startLine < len(lines) && isMarkdown && f.config.Files.PlacementExceptions.MarkdownHeading && hasMarkdownHeading(lines[startLine:]) {
		result = append(result, lines[startLine])
		startLine++
	}

	// Determine scan limit for header area
	maxScan := len(lines)
	if f.config.Detection.MaxScanLines > 0 {
		maxScan = min(startLine+f.config.Detection.MaxScanLines, len(lines))
	}

	// Get comment prefix for SPDX detection
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	commentPrefix := f.config.Files.CommentStyles[extKey]
	if commentPrefix == "" {
		// Fallback to hardcoded values if not found in config
		switch ext {
		case ".go":
			commentPrefix = "//"
		case ".sh", ".py", ".hcl", ".tf", ".yml", ".yaml":
			commentPrefix = "#"
		case ".md", ".html.markdown":
			commentPrefix = "<!--"
		default:
			commentPrefix = "//"
		}
	}

	// Scan for existing copyrights and third-party copyrights in header area only
	hasCorrectCopyright := false
	hasCorrectLicense := false
	for i := startLine; i < maxScan; i++ {
		line := lines[i]
		if f.config.ShouldReplace(line) {
			hasCopyright = true
		} else if f.config.IsOwnCopyrightLine(line, ext) {
			// Found our own copyright line - mark for replacement if not current
			if strings.TrimSpace(line) != strings.TrimSpace(copyrightHeader) {
				hasCopyright = true
			} else {
				hasCorrectCopyright = true
			}
		} else if f.config.IsThirdPartyCopyright(line) {
			thirdPartyLines = append(thirdPartyLines, line)
		} else if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) {
			hasCorrectCopyright = true
		} else if licenseHeader != "" && strings.TrimSpace(line) == strings.TrimSpace(licenseHeader) {
			hasCorrectLicense = true
		} else if isSPDXHeaderLine(line, commentPrefix) {
			// Found an SPDX header line - we'll need to replace it if it's not exactly our format
			if licenseHeader == "" || strings.TrimSpace(line) != strings.TrimSpace(licenseHeader) {
				hasCopyright = true // Mark as needing replacement
			}
		}
	}

	// If both copyright and license (if enabled) are already correct, nothing to do
	if hasCorrectCopyright && (licenseHeader == "" || hasCorrectLicense) {
		return false
	}

	// Helper to add copyright headers with proper block comment wrapping
	addHeaders := func(r *[]string) {
		if isBlockCommentStyle(f.config, ext) {
			*r = append(*r, "/**")
			*r = append(*r, copyrightHeader)
			if licenseHeader != "" {
				*r = append(*r, licenseHeader)
			}
			*r = append(*r, " */")
		} else {
			*r = append(*r, copyrightHeader)
			if licenseHeader != "" {
				*r = append(*r, licenseHeader)
			}
		}
	}

	// Handle third-party copyrights based on action
	switch f.config.ThirdParty.Action {
	case "above":
		// Add our copyright above third-party
		addHeaders(&result)
		result = append(result, thirdPartyLines...)
		addBlankLineIfNeeded(&result, lines, startLine)
	case "below":
		// Add third-party first, then our copyright
		result = append(result, thirdPartyLines...)
		addHeaders(&result)
		addBlankLineIfNeeded(&result, lines, startLine)
	case "replace":
		// Replace third-party with our copyright
		addHeaders(&result)
		addBlankLineIfNeeded(&result, lines, startLine)
	default: // "leave" or unspecified
		// Just add our copyright, leave third-party as-is
		addHeaders(&result)
		addBlankLineIfNeeded(&result, lines, startLine)
	}

	// Process remaining content, only removing copyrights from header area
	skipNext := false
	inCopyrightBlock := false // Track if we're inside a multi-line comment with copyright
	skipNextBlank := false    // Skip blank line after copyright block removal
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		inHeaderArea := i < maxScan

		// Only skip/remove copyright lines if in header area
		if inHeaderArea {
			// Detect start of multi-line comment block (<!-- or /**)
			if trimmed == "<!--" || trimmed == "/**" {
				closeMarker := "-->"
				if trimmed == "/**" {
					closeMarker = "*/"
				}
				// Look ahead to see if this block contains copyright
				for j := i + 1; j < maxScan && j < len(lines); j++ {
					checkLine := lines[j]
					checkTrimmed := strings.TrimSpace(checkLine)
					if checkTrimmed == closeMarker || strings.HasSuffix(checkTrimmed, closeMarker) {
						break
					}
					if f.config.ShouldReplace(checkLine) || f.config.IsOwnCopyrightLine(checkLine, ext) || isSPDXHeaderLine(checkLine, commentPrefix) {
						inCopyrightBlock = true
						fixed = true
						break
					}
				}
				if inCopyrightBlock {
					continue // Skip the opening marker
				}
			}

			// Skip lines inside a copyright block
			if inCopyrightBlock {
				if trimmed == "-->" || trimmed == "*/" || strings.HasSuffix(trimmed, "*/") {
					inCopyrightBlock = false
					skipNextBlank = true
				}
				continue
			}

			// Skip blank line after copyright block removal
			if skipNextBlank && trimmed == "" {
				skipNextBlank = false
				continue
			}
			skipNextBlank = false

			// Remove old copyright/license lines if we're adding new ones
			if strings.TrimSpace(line) == strings.TrimSpace(copyrightHeader) ||
				(licenseHeader != "" && strings.TrimSpace(line) == strings.TrimSpace(licenseHeader)) {
				skipNext = true
				continue
			}

			// Remove our own copyright lines that need updating
			if f.config.IsOwnCopyrightLine(line, ext) {
				fixed = true
				skipNext = true
				continue
			}

			// Remove any SPDX header line (handles duplicates and different formats)
			if isSPDXHeaderLine(line, commentPrefix) {
				fixed = true
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
		_ = os.WriteFile(file, []byte(newContent), 0644)
		return true
	}

	return false
}

// addBlankLineIfNeeded adds a blank line only if the next content line isn't already blank
func addBlankLineIfNeeded(result *[]string, lines []string, startLine int) {
	// Check if the next line to be processed is blank
	nextLineIsBlank := startLine < len(lines) && strings.TrimSpace(lines[startLine]) == ""

	// Only add blank line if next line isn't already blank
	if !nextLineIsBlank {
		*result = append(*result, "")
	}
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

	// Get comment prefix for SPDX detection
	extKey := strings.TrimPrefix(ext, ".")
	extKey = strings.ReplaceAll(extKey, ".", "_")
	commentPrefix := f.config.Files.CommentStyles[extKey]
	if commentPrefix == "" {
		// Fallback to hardcoded values if not found in config
		switch ext {
		case ".go":
			commentPrefix = "//"
		case ".sh", ".py", ".hcl", ".tf", ".yml", ".yaml":
			commentPrefix = "#"
		case ".md", ".html.markdown":
			commentPrefix = "<!--"
		default:
			commentPrefix = "//"
		}
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

			// Remove our own copyright lines that need updating
			if f.config.IsOwnCopyrightLine(line, ext) {
				skipNext = true
				continue
			}

			// Remove any SPDX header line (handles duplicates and different formats)
			if isSPDXHeaderLine(line, commentPrefix) {
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
