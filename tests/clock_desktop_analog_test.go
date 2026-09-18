package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func clockPath(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	candidates := []string{
		filepath.Join(root, "bin", "clock.exe"),
		filepath.Join(root, "bin", "clock"),
	}
	for _, bin := range candidates {
		if _, err := os.Stat(bin); err == nil {
			return bin
		}
	}
	t.Fatalf("bin/clock not found under %s (run ./scripts/process/build.sh first)", filepath.Join(root, "bin"))
	return ""
}

func TestClockDesktopAnalog_BinaryExists(t *testing.T) {
	_ = clockPath(t)
}

func TestClockDesktopAnalog_IndependentFromMyprog(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	modPath := filepath.Join(root, "features", "clock", "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if bytes.Contains(data, []byte("features/myprog")) {
		t.Fatalf("features/clock/go.mod must not depend on myprog")
	}

	err = filepath.Walk(filepath.Join(root, "features", "clock"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(src, []byte("github.com/axsh/tokotachi/features/myprog")) {
			t.Errorf("%s imports myprog", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk clock sources: %v", err)
	}
}

func TestClockDesktopAnalog_SmokeQuitAfter(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("clock GUI smoke test requires Windows")
	}

	bin := clockPath(t)
	cmd := exec.Command(bin, "-quit-after", "2s")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start clock: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("clock exited with error: %v; stderr=%q stdout=%q", err, stderr.String(), stdout.String())
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("clock did not exit within 10s; stderr=%q", stderr.String())
	}

	if strings.Contains(stderr.String(), "ERROR: clock exited:") {
		t.Fatalf("unexpected ERROR in stderr: %q", stderr.String())
	}
}
