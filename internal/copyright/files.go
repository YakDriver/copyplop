// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package copyright

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/YakDriver/copyplop/internal/config"
)

func getTrackedFiles(path string, cfg *config.Config) ([]string, error) {
	if cfg.Files.GitTracked {
		return getGitFiles(path)
	}
	return getAllFiles(path)
}

func getGitFiles(path string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for line := range strings.SplitSeq(string(output), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func getAllFiles(path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func hasShebang(lines []string) bool {
	return len(lines) > 0 && strings.HasPrefix(lines[0], "#!")
}

func getFrontmatterEnd(lines []string, cfg *config.Config, file string) int {
	// Check for compound extensions (e.g., .html.markdown)
	for _, belowExt := range cfg.Files.BelowFrontmatter {
		if strings.HasSuffix(file, belowExt) {
			if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
				// Find closing ---
				for i := 1; i < len(lines); i++ {
					if strings.TrimSpace(lines[i]) == "---" {
						return i + 1 // Return line after closing ---
					}
				}
			}
			break
		}
	}
	return 0 // No frontmatter or not configured for this extension
}
