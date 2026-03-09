package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	// back up original and restore afterwards
	orig := Version
	defer func() { Version = orig }()

	Version = "v1.2.3-test"

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != Version {
		t.Fatalf("expected version %q, got %q", Version, output)
	}
}
