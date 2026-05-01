package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrate_HasRedactFlags(t *testing.T) {
    // Build the migrate binary in a temp location, then run it with --help.
    tmpMigrate := filepath.Join(os.TempDir(), "migrate_test_bin")
    build := exec.Command("go", "build", "-o", tmpMigrate, "./cmd/migrate")
    build.Dir = "../../"
    if b, err := build.CombinedOutput(); err != nil {
        t.Fatalf("failed to build migrate binary: %v\noutput: %s", err, string(b))
    }
    defer os.Remove(tmpMigrate)
    cmd := exec.Command(tmpMigrate, "--help")
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to run migrate --help: %v\noutput: %s", err, string(out))
    }
    s := string(out)
    if !strings.Contains(s, "redact_enabled") {
        t.Fatalf("migrate help missing redact_enabled flag: %s", s)
    }
    if !strings.Contains(s, "redact_request_body") {
        t.Fatalf("migrate help missing redact_request_body flag: %s", s)
    }
}

func TestSeed_HasRedactFlags(t *testing.T) {
    // Run `go run ./cmd/seed --help` and inspect output even if the process
    // exits with non-zero status (some CLIs print usage then exit non-zero).
    // Build the seed binary and run it with --help
    tmpSeed := filepath.Join(os.TempDir(), "seed_test_bin")
    buildSeed := exec.Command("go", "build", "-o", tmpSeed, "./cmd/seed")
    buildSeed.Dir = "../../"
    if b, err := buildSeed.CombinedOutput(); err != nil {
        t.Fatalf("failed to build seed binary: %v\noutput: %s", err, string(b))
    }
    defer os.Remove(tmpSeed)
    cmd := exec.Command(tmpSeed, "--help")
    cmd.Env = append(cmd.Env, "GOTRACEBACK=all")
    out, _ := cmd.CombinedOutput()
    s := string(out)
    if !strings.Contains(s, "redact_enabled") {
        t.Fatalf("seed help missing redact_enabled flag: %s", s)
    }
    if !strings.Contains(s, "redact_request_body") {
        t.Fatalf("seed help missing redact_request_body flag: %s", s)
    }
}
