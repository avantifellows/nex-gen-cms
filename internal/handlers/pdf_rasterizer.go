package handlers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// pdfToImages rasterises every page of pdfBytes to PNG at the given DPI,
// returning one []byte per page sorted by page number.
//
// It probes for rasterisers in this order:
//  1. pdftoppm  (Poppler)       – winget install Poppler / apt install poppler-utils
//  2. mutool    (MuPDF CLI)     – winget install MuPDF  / apt install mupdf-tools
//  3. gswin64c / gs (Ghostscript) – winget install Ghostscript.Ghostscript
func pdfToImages(pdfBytes []byte, dpi int) ([][]byte, error) {
	tmp, err := os.MkdirTemp("", "pdf-raster-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	pdfPath := filepath.Join(tmp, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0o644); err != nil {
		return nil, fmt.Errorf("writing temp PDF: %w", err)
	}

	type rasterFn func(string, string, int) ([][]byte, error)
	attempts := []struct {
		name string
		fn   rasterFn
	}{
		{"pdftoppm", rasterWithPdftoppm},
		{"mutool", rasterWithMutool},
		{"ghostscript", rasterWithGhostscript},
	}

	var lastErr error
	for _, a := range attempts {
		imgs, err := a.fn(pdfPath, tmp, dpi)
		if err == nil {
			return imgs, nil
		}
		lastErr = fmt.Errorf("%s: %w", a.name, err)
	}

	return nil, fmt.Errorf(
		"no PDF rasteriser found (%v). "+
			"Install one of: pdftoppm (poppler-utils), mutool (mupdf-tools), or Ghostscript. "+
			"Windows: `winget install Poppler` or `winget install Ghostscript.Ghostscript`",
		lastErr,
	)
}

func rasterWithPdftoppm(pdfPath, outDir string, dpi int) ([][]byte, error) {
	bin, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf("not found in PATH")
	}
	prefix := filepath.Join(outDir, "p")
	var stderr bytes.Buffer
	cmd := exec.Command(bin, "-r", strconv.Itoa(dpi), "-png", pdfPath, prefix)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run: %v — %s", err, stderr.String())
	}
	return collectPNGs(outDir, "p")
}

func rasterWithMutool(pdfPath, outDir string, dpi int) ([][]byte, error) {
	bin, err := exec.LookPath("mutool")
	if err != nil {
		return nil, fmt.Errorf("not found in PATH")
	}
	outPattern := filepath.Join(outDir, "p-%d.png")
	var stderr bytes.Buffer
	cmd := exec.Command(bin, "draw", "-r", strconv.Itoa(dpi), "-o", outPattern, pdfPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run: %v — %s", err, stderr.String())
	}
	return collectPNGs(outDir, "p-")
}

func rasterWithGhostscript(pdfPath, outDir string, dpi int) ([][]byte, error) {
	bin := "gs"
	for _, candidate := range []string{"gswin64c", "gswin32c", "gs"} {
		if _, err := exec.LookPath(candidate); err == nil {
			bin = candidate
			break
		}
	}
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("not found in PATH")
	}
	outPattern := filepath.Join(outDir, "p-%03d.png")
	var stderr bytes.Buffer
	cmd := exec.Command(bin,
		"-dNOPAUSE", "-dBATCH", "-dSAFER",
		"-sDEVICE=png16m",
		fmt.Sprintf("-r%d", dpi),
		"-sOutputFile="+outPattern,
		pdfPath,
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run: %v — %s", err, stderr.String())
	}
	return collectPNGs(outDir, "p-")
}

// collectPNGs gathers *.png files whose names start with prefix, sorted
// numerically by the trailing integer so pages stay in order.
func collectPNGs(dir, prefix string) ([][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type numbered struct {
		n    int
		path string
	}
	var files []numbered
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".png") {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		base := strings.TrimPrefix(name, prefix)
		base = strings.TrimSuffix(base, ".png")
		base = strings.TrimLeft(base, "-")
		n, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		files = append(files, numbered{n, filepath.Join(dir, name)})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no PNG files with prefix %q generated in %s", prefix, dir)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].n < files[j].n })

	imgs := make([][]byte, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.path, err)
		}
		imgs = append(imgs, data)
	}
	return imgs, nil
}
