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
	files, err := getTrackedFiles(path)
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
		bar.Add(1)
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

	startLine := 0
	if hasShebang(lines) {
		startLine = 1
	}

	if startLine >= len(lines) {
		return &Issue{File: file, Problem: "missing copyright header"}
	}

	if !strings.Contains(lines[startLine], strings.TrimSpace(expectedHeader[2:])) {
		return &Issue{File: file, Problem: "missing or incorrect copyright header"}
	}

	return nil
}
