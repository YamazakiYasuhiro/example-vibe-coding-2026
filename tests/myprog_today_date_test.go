package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func myprogPath(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	// Windows CreateProcess requires a .exe suffix; build.sh emits GOEXE when applicable.
	candidates := []string{
		filepath.Join(root, "bin", "myprog.exe"),
		filepath.Join(root, "bin", "myprog"),
	}
	for _, bin := range candidates {
		if _, err := os.Stat(bin); err == nil {
			return bin
		}
	}
	t.Fatalf("bin/myprog not found under %s (run ./scripts/process/build.sh first)", filepath.Join(root, "bin"))
	return ""
}

func runMyprog(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(myprogPath(t), args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		return stdout, stderr, 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return stdout, stderr, ee.ExitCode()
	}
	t.Fatalf("run myprog: %v", err)
	return "", "", -1
}

func assertHelloAndDate(t *testing.T, stdout, wantDate string) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdout lines = %d, want 2; stdout=%q", len(lines), stdout)
	}
	if lines[0] != "Hello, World!" {
		t.Fatalf("stdout line1 = %q, want %q", lines[0], "Hello, World!")
	}
	if lines[1] != wantDate {
		t.Fatalf("stdout line2 = %q, want %q", lines[1], wantDate)
	}
}

func TestMyprogTodayDate_Default(t *testing.T) {
	stdout, stderr, code := runMyprog(t)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	assertHelloAndDate(t, stdout, time.Now().Format("2006-01-02"))
}

func TestMyprogTodayDate_FormatIso(t *testing.T) {
	stdout, stderr, code := runMyprog(t, "-format", "iso")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	assertHelloAndDate(t, stdout, time.Now().Format("2006-01-02"))
}

func TestMyprogTodayDate_FormatSlash(t *testing.T) {
	stdout, stderr, code := runMyprog(t, "-format", "slash")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	assertHelloAndDate(t, stdout, time.Now().Format("2006/01/02"))
}

func TestMyprogTodayDate_FormatJp(t *testing.T) {
	stdout, stderr, code := runMyprog(t, "-format", "jp")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	assertHelloAndDate(t, stdout, time.Now().Format("2006年01月02日"))
}

func TestMyprogTodayDate_InvalidFormat(t *testing.T) {
	stdout, stderr, code := runMyprog(t, "-format", "unknown")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr, "unsupported format") {
		t.Fatalf("stderr = %q, want substring %q", stderr, "unsupported format")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty on invalid format", stdout)
	}
}
