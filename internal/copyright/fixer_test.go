// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package copyright

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YakDriver/copyplop/internal/config"
)

func TestFixer_fixFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Copyright: config.Copyright{
			Holder:      "IBM Corp.",
			StartYear:   2014,
			CurrentYear: 2025,
			Format:      "Copyright {{.Holder}} {{.StartYear}}, {{.CurrentYear}}",
		},
		License: config.License{
			Enabled:    true,
			Identifier: "MPL-2.0",
			Format:     "SPDX-License-Identifier: {{.Identifier}}",
		},
		Files: config.Files{
			CommentStyles: map[string]string{".go": "//", ".sh": "#"},
		},
		Detection: config.Detection{
			SkipGenerated:     true,
			GeneratedPatterns: []string{"Code generated"},
			ReplacePatterns:   []string{"Copyright.*HashiCorp"},
			MaxScanLines:      20,
			RequireAtTop:      true,
		},
		ThirdParty: config.ThirdParty{
			Action:   "above",
			Patterns: []string{"Copyright.*Oracle"},
		},
	}

	fixer := NewFixer(cfg)

	tests := []struct {
		name           string
		filename       string
		input          string
		expectedOutput string
		shouldFix      bool
	}{
		{
			name:     "replace HashiCorp header",
			filename: "replace.go",
			input: `// Copyright HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			shouldFix: true,
		},
		{
			name:     "add missing header",
			filename: "missing.go",
			input: `package main

func main() {}`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			shouldFix: true,
		},
		{
			name:     "third-party copyright - above action",
			filename: "oracle.go",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			shouldFix: true,
		},
		{
			name:     "already correct header",
			filename: "correct.go",
			input: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			shouldFix: false,
		},
		{
			name:     "copyright deep in file beyond scan limit",
			filename: "deep.go",
			input: `package main

// Line 3
// Line 4
// Line 5
// Line 6
// Line 7
// Line 8
// Line 9
// Line 10
// Line 11
// Line 12
// Line 13
// Line 14
// Line 15
// Line 16
// Line 17
// Line 18
// Line 19
// Line 20
// Line 21
// Line 22
// Line 23
// Line 24
// Line 25
// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

func main() {}`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main

// Line 3
// Line 4
// Line 5
// Line 6
// Line 7
// Line 8
// Line 9
// Line 10
// Line 11
// Line 12
// Line 13
// Line 14
// Line 15
// Line 16
// Line 17
// Line 18
// Line 19
// Line 20
// Line 21
// Line 22
// Line 23
// Line 24
// Line 25
// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

func main() {}`,
			shouldFix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			fixed := fixer.fixFile(filePath)
			if fixed != tt.shouldFix {
				t.Errorf("fixFile() fixed = %v, want %v", fixed, tt.shouldFix)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file after fix: %v", err)
			}

			result := string(content)
			if result != tt.expectedOutput {
				t.Errorf("File content mismatch:\nGot:\n%s\nWant:\n%s", result, tt.expectedOutput)
			}
		})
	}
}

func TestThirdPartyActions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		action         string
		input          string
		expectedOutput string
	}{
		{
			name:   "below action",
			action: "below",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.
// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
		},
		{
			name:   "replace action",
			action: "replace",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
		},
		{
			name:   "leave action",
			action: "leave",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Copyright: config.Copyright{
					Holder:      "IBM Corp.",
					StartYear:   2014,
					CurrentYear: 2025,
					Format:      "Copyright {{.Holder}} {{.StartYear}}, {{.CurrentYear}}",
				},
				License: config.License{
					Enabled:    true,
					Identifier: "MPL-2.0",
					Format:     "SPDX-License-Identifier: {{.Identifier}}",
				},
				Files: config.Files{
					CommentStyles: map[string]string{".go": "//"},
				},
				ThirdParty: config.ThirdParty{
					Action:   tt.action,
					Patterns: []string{"Copyright.*Oracle"},
				},
			}

			fixer := NewFixer(cfg)
			filePath := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(filePath, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			fixer.fixFile(filePath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file after fix: %v", err)
			}

			result := string(content)
			if result != tt.expectedOutput {
				t.Errorf("Action %s failed:\nGot:\n%s\nWant:\n%s", tt.action, result, tt.expectedOutput)
			}
		})
	}
}

func FuzzCopyplopNormalize(f *testing.F) {
	// Test cases for different file types with appropriate extensions and content
	testCases := []struct {
		ext     string
		comment string
		seeds   []string
	}{
		{
			ext:     ".go",
			comment: "//",
			seeds: []string{
				"// Copyright X\n// SPDX-License-Identifier: Apache-2.0\n\npackage main\n",
				"package main\n",
				"\n\n// Copyright Old\n\npackage main\n",
				"// SPDX-License-Identifier: MIT\npackage main\n",
				"\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n// Copyright X\n// SPDX-License-Identifier: Apache-2.0\n\npackage main\n",
				"// Code generated by some tool; DO NOT EDIT.\n\npackage main\n",
				"Copyright 2025 Dirk Avery\n\npackage main\n",
				"Copyright Dirk Avery 2025\n\npackage main\n",
				"// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0\n\npackage main\n",
			},
		},
		{
			ext:     ".sh",
			comment: "#",
			seeds: []string{
				"#!/bin/bash\n# Copyright X\n# SPDX-License-Identifier: Apache-2.0\n\necho hello\n",
				"#!/bin/bash\necho hello\n",
				"# Copyright Old\necho hello\n",
			},
		},
		{
			ext:     ".py",
			comment: "#",
			seeds: []string{
				"#!/usr/bin/env python3\n# Copyright X\n# SPDX-License-Identifier: MIT\n\nprint('hello')\n",
				"print('hello')\n",
				"# Copyright Old\nprint('hello')\n",
			},
		},
	}

	// Add seeds for each file type
	for _, tc := range testCases {
		for _, seed := range tc.seeds {
			f.Add(seed, tc.ext)
		}
	}

	f.Fuzz(func(t *testing.T, s string, ext string) {
		if len(s) > 100_000 {
			t.Skip()
		}

		// Skip invalid extensions
		if ext != ".go" && ext != ".sh" && ext != ".py" {
			t.Skip()
		}

		// Create appropriate config for the extension
		cfg := createConfigForExtension(ext)
		fixer := NewFixer(cfg)

		// Get the actual canonical headers from config
		canonicalCopyright, _ := cfg.GetCopyrightHeader(ext)
		canonicalSPDX, _ := cfg.GetLicenseHeader(ext)

		// Use the real copyplop logic
		out1, err := fixer.ProcessContent([]byte(s), ext)
		if err != nil {
			t.Fatalf("ProcessContent error: %v", err)
		}

		// Property 1: idempotence
		out2, err := fixer.ProcessContent(out1, ext)
		if err != nil {
			t.Fatalf("ProcessContent second run error: %v", err)
		}
		if string(out1) != string(out2) {
			t.Fatalf("not idempotent:\nfirst:\n%s\n\nsecond:\n%s", out1, out2)
		}

		// Property 2: canonical header present
		outStr := string(out1)
		if !hasCanonicalHeaderDynamic(outStr, canonicalCopyright, canonicalSPDX) {
			t.Fatalf("missing canonical header:\n%s", outStr)
		}

		// Property 3: body preserved (allowing for blank line normalization around headers)
		if !bodiesEquivalent(s, outStr) {
			t.Fatalf("body content changed unexpectedly\nOriginal:\n%q\nProcessed:\n%q", s, outStr)
		}
	})
}

// createConfigForExtension creates a realistic config for the given extension
func createConfigForExtension(ext string) *config.Config {
	var commentStyle string
	switch ext {
	case ".go":
		commentStyle = "//"
	case ".sh", ".py":
		commentStyle = "#"
	default:
		commentStyle = "//"
	}

	return &config.Config{
		Copyright: config.Copyright{
			Holder:      "Dirk Avery",
			CurrentYear: 2025,
			Format:      "Copyright {{.Holder}} {{.CurrentYear}}",
		},
		License: config.License{
			Enabled:    true,
			Identifier: "MIT",
			Format:     "SPDX-License-Identifier: {{.Identifier}}",
		},
		Files: config.Files{
			Extensions:    []string{".go", ".sh", ".py"},
			CommentStyles: map[string]string{ext: commentStyle},
		},
		Detection: config.Detection{
			ReplacePatterns: []string{"Copyright (c) HashiCorp, Inc."},
			MaxScanLines:    20,
		},
		ThirdParty: config.ThirdParty{
			Action:   "replace",
			Patterns: []string{"Copyright.*"},
		},
	}
}

func hasCanonicalHeaderDynamic(s, copyright, spdx string) bool {
	lines := strings.Split(s, "\n")

	// Skip shebang if present
	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#!") {
		startIdx = 1
	}

	// Need at least 3 lines after shebang: copyright, spdx, blank
	if len(lines) < startIdx+3 {
		return false
	}

	if lines[startIdx] != copyright {
		return false
	}
	if lines[startIdx+1] != spdx {
		return false
	}
	if lines[startIdx+2] != "" {
		return false
	}
	return true
}

// extractRealContent extracts the meaningful content, ignoring headers and leading whitespace
func extractRealContent(s string) string {
	lines := strings.Split(s, "\n")
	var contentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip obvious header lines
		if strings.HasPrefix(trimmed, "#!") ||
			strings.Contains(trimmed, "Copyright") ||
			strings.Contains(trimmed, "SPDX") ||
			(strings.HasPrefix(trimmed, "//") && (strings.Contains(trimmed, "License") || len(trimmed) < 10)) {
			continue
		}

		// Skip blank lines at the start
		if len(contentLines) == 0 && trimmed == "" {
			continue
		}

		contentLines = append(contentLines, line)
	}

	result := strings.Join(contentLines, "\n")
	// Normalize trailing newlines - if original had trailing newline, preserve it
	if strings.HasSuffix(s, "\n") && !strings.HasSuffix(result, "\n") && len(contentLines) > 0 {
		result += "\n"
	}
	return result
}

func TestDebugBlankLines(t *testing.T) {
	cfg := &config.Config{
		Copyright: config.Copyright{
			Holder:      "Dirk Avery",
			StartYear:   2025,
			CurrentYear: 2025,
			Format:      "Copyright (c) {{.CurrentYear}} {{.Holder}}",
		},
		License: config.License{
			Enabled:    true,
			Identifier: "MIT",
			Format:     "SPDX-License-Identifier: {{.Identifier}}",
		},
		Files: config.Files{
			CommentStyles: map[string]string{".go": "//"},
		},
		Detection: config.Detection{
			ReplacePatterns: []string{"Copyright.*"},
			MaxScanLines:    20,
		},
		ThirdParty: config.ThirdParty{
			Action:   "replace",
			Patterns: []string{"Copyright.*"},
		},
	}

	fixer := NewFixer(cfg)

	input := "0\n\n0"
	t.Logf("Input: %q", input)

	out, err := fixer.ProcessContent([]byte(input), ".go")
	if err != nil {
		t.Fatalf("ProcessContent error: %v", err)
	}

	t.Logf("Output: %q", string(out))

	// Check what extractRealContent does
	bodyIn := extractRealContent(input)
	bodyOut := extractRealContent(string(out))

	t.Logf("Body IN: %q", bodyIn)
	t.Logf("Body OUT: %q", bodyOut)
}

// bodiesEquivalent checks if the meaningful content is preserved, allowing for
// blank line normalization around copyright headers
func bodiesEquivalent(original, processed string) bool {
	// Extract non-header content from both
	origContent := extractMeaningfulContent(original)
	procContent := extractMeaningfulContent(processed)

	// Compare the meaningful content tokens
	return contentTokensEqual(origContent, procContent)
}

// extractMeaningfulContent gets the actual code/content, skipping headers and normalizing whitespace
func extractMeaningfulContent(s string) []string {
	lines := strings.Split(s, "\n")
	var content []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip header-like lines
		if strings.HasPrefix(trimmed, "#!") ||
			strings.Contains(trimmed, "Copyright") ||
			strings.Contains(trimmed, "SPDX") ||
			(strings.HasPrefix(trimmed, "//") && (strings.Contains(trimmed, "License") || len(trimmed) < 10)) {
			continue
		}

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		content = append(content, trimmed)
	}

	return content
}

// contentTokensEqual compares content tokens, ignoring whitespace differences
func contentTokensEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
