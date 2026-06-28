// Package paddleocrpy provides OCR via the local PaddleOCR Python library.
// It calls a helper Python script that uses PaddleOCR with:
//   - enable_mkldnn=False (required on Windows/AMD)
//   - Automatic slicing for images taller than maxHeight (avoids resize distortion)
//
// This bypasses the VLM/API approach entirely and runs OCR directly on the machine.
package paddleocrpy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/schaepher/ocr/document"
)

const defaultMaxHeight = 3800
const defaultOverlap = 200

// Run runs PaddleOCR Python on the given image and returns a structured Document.
// maxHeight controls the slice height (0 = use default 3800).
// overlap controls the vertical overlap between slices (0 = use default 200).
// onProgress is called for each slice: (current, total, yOffset).
func Run(ctx context.Context, imagePath string, maxHeight, overlap int, onProgress func(cur, total int, y int)) (*document.Document, error) {
	if maxHeight <= 0 {
		maxHeight = defaultMaxHeight
	}
	if overlap <= 0 {
		overlap = defaultOverlap
	}

	scriptPath, err := findHelperScript()
	if err != nil {
		return nil, fmt.Errorf("paddleocrpy: find helper: %w", err)
	}

	args := []string{
		scriptPath,
		"--image", imagePath,
		"--max-height", strconv.Itoa(maxHeight),
		"--overlap", strconv.Itoa(overlap),
	}

	cmd := exec.CommandContext(ctx, pythonExe(), args...)
	cmd.Stderr = &progressParser{onProgress: onProgress}

	start := time.Now()
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("paddleocrpy: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("paddleocrpy: %w", err)
	}
	_ = time.Since(start)

	var doc document.Document
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("paddleocrpy: parse output: %w\nraw: %s", err, string(out[:min(len(out), 500)]))
	}

	return &doc, nil
}

// progressParser extracts progress info from Python stderr (e.g. "slice [1/6] y=0-4000: 34 texts").
type progressParser struct {
	onProgress func(cur, total int, y int)
}

func (p *progressParser) Write(data []byte) (int, error) {
	if p.onProgress != nil {
		line := strings.TrimSpace(string(data))
		// Parse: "slice [1/6] y=0-4000: 34 texts"
		if strings.HasPrefix(line, "slice [") {
			parts := strings.SplitN(line, "]", 2)
			if len(parts) == 2 {
				idxTotal := strings.TrimPrefix(parts[0], "slice [")
				nums := strings.SplitN(idxTotal, "/", 2)
				if len(nums) == 2 {
					cur, _ := strconv.Atoi(nums[0])
					total, _ := strconv.Atoi(nums[1])
					// Extract y offset from "y=0-4000: ..."
					rest := parts[1]
					if strings.HasPrefix(rest, " y=") {
						rest = rest[3:] // Remove " y="
						if dash := strings.IndexByte(rest, '-'); dash > 0 {
							y, _ := strconv.Atoi(rest[:dash])
							p.onProgress(cur, total, y)
						}
					}
				}
			}
		}
	}
	// Also write to os.Stderr for user visibility
	os.Stderr.Write(data)
	return len(data), nil
}

// pythonExe returns the Python executable path.
// Checks common virtual environment locations relative to CWD.
func pythonExe() string {
	// Check PADDLEOCR_PYTHON_EXE env var first
	if env := os.Getenv("PADDLEOCR_PYTHON_EXE"); env != "" {
		if _, err := os.Stat(env); err == nil {
			return env
		}
	}

	py := "python3"
	if runtime.GOOS == "windows" {
		py = "python"
	}

	// Check common venv paths relative to CWD
	if wd, err := os.Getwd(); err == nil {
		for _, venv := range []string{
			filepath.Join(wd, "paddleocr_env", "Scripts", py+".exe"),
			filepath.Join(wd, "paddleocr_env", "bin", py),
			filepath.Join(wd, "venv", "Scripts", py+".exe"),
			filepath.Join(wd, "venv", "bin", py),
			filepath.Join(wd, ".venv", "Scripts", py+".exe"),
			filepath.Join(wd, ".venv", "bin", py),
		} {
			if _, err := os.Stat(venv); err == nil {
				return venv
			}
		}
	}

	return py
}

// findHelperScript locates ocr_helper.py relative to this source file or the executable.
func findHelperScript() (string, error) {
	// Look relative to the module: provider/paddleocrpy/ocr_helper.py
	// Try multiple strategies.
	candidates := []string{}

	// 1. Relative to the current working directory (development)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "provider", "paddleocrpy", "ocr_helper.py"),
		)
	}

	// 2. Relative to the executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "provider", "paddleocrpy", "ocr_helper.py"),
			filepath.Join(exeDir, "ocr_helper.py"),
		)
	}

	// Also check PADDLEOCR_PYTHON_HELPER env var
	if env := os.Getenv("PADDLEOCR_PYTHON_HELPER"); env != "" {
		candidates = append([]string{env}, candidates...)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return filepath.Abs(p)
		}
	}

	return "", fmt.Errorf("ocr_helper.py not found in: %v", candidates)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
