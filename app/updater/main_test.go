package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// Run the real main entry point from a copied test executable. A current
// throttle stamp prevents network access; the legacy -watch argument must
// neither keep the updater alive nor create/append diagnostic files.
func TestThrottledUpdaterDoesNotWaitOrWriteDiagnostics(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(strconv.FormatBool(existing), func(t *testing.T) {
			dir := t.TempDir()
			self, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			binary, err := os.ReadFile(self)
			if err != nil {
				t.Fatal(err)
			}
			helper := filepath.Join(dir, "updater-check.exe")
			if err := os.WriteFile(helper, binary, 0755); err != nil {
				t.Fatal(err)
			}
			stamp := filepath.Join(dir, "update-stamp.txt")
			if err := os.WriteFile(stamp, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(dir, "crash.log")
			sentinel := []byte("existing user diagnostics\n")
			if existing {
				if err := os.WriteFile(logPath, sentinel, 0644); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, helper, "-test.run=^TestUpdaterHelper$", "--", "-watch", strconv.Itoa(os.Getpid()))
			cmd.Env = append(os.Environ(), "TUIGO_UPDATER_TEST_HELPER=1")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("updater should exit without waiting: %v %s", err, output)
			}
			got, err := os.ReadFile(logPath)
			if existing {
				if err != nil || !bytes.Equal(got, sentinel) {
					t.Fatalf("existing log changed: %v %q", err, got)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("updater created a diagnostic file: %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if existing {
				want++
			}
			if len(entries) != want {
				t.Fatalf("unexpected files after updater exit: %v", entries)
			}
		})
	}
}

func TestUpdaterHelper(t *testing.T) {
	if os.Getenv("TUIGO_UPDATER_TEST_HELPER") != "1" {
		t.Skip("subprocess entry only")
	}
	main()
}
