package cli

import "testing"

func TestVersion(t *testing.T) {
	res := runCommand(t, &fakeRuntime{}, "version")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	if res.stdout != "lightrdp v0.0.1\n" {
		t.Fatalf("Expected 'lightrdp v0.0.1', got: %q", res.stdout)
	}
	if ExitCode(res.err) != exitOK {
		t.Fatalf("Expected exit code %d, got: %d", exitOK, ExitCode(res.err))
	}
}
