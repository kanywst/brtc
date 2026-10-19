package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestReadStdinPassword(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "echo adds a trailing newline", in: "P@ssw0rd123!\n", want: "P@ssw0rd123!"},
		{name: "printf adds none", in: "P@ssw0rd123!", want: "P@ssw0rd123!"},
		{name: "CRLF line ending", in: "P@ssw0rd123!\r\n", want: "P@ssw0rd123!"},
		{name: "inner spaces are kept", in: "correct horse battery staple\n", want: "correct horse battery staple"},
		{name: "empty input", in: "", want: ""},
		// Analyzed as one 30-character password, these four used to pass a
		// --fail-under-time 1y gate that each of them fails on its own.
		{name: "a file of passwords", in: "123456\npassword\nqwerty\nletmein\n", wantErr: true},
		{name: "CR-separated lines", in: "123456\rpassword", wantErr: true},
		{name: "blank line between", in: "123456\n\npassword\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readStdinPassword(strings.NewReader(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("readStdinPassword(%q) = %q, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("readStdinPassword(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("readStdinPassword(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// A read failure must surface rather than fall through to the
// "password is required" message, which would hide the cause.
func TestReadStdinPassword_ReadError(t *testing.T) {
	if _, err := readStdinPassword(failingReader{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want the read error to surface, got %v", err)
	}
}
