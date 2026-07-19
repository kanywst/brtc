package breach

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHashParts_KnownSHA1(t *testing.T) {
	// SHA-1("password") = 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8 (uppercase).
	prefix, suffix := hashParts("password")
	if prefix != "5BAA6" {
		t.Errorf("prefix = %q, want 5BAA6", prefix)
	}
	if suffix != "1E4C9B93F3F0682250B6CF8331B7EE68FD8" {
		t.Errorf("suffix = %q, unexpected", suffix)
	}
}

func TestCountInBody(t *testing.T) {
	body := strings.Join([]string{
		"0000000000000000000000000000000000A:0", // padding
		"1E4C9B93F3F0682250B6CF8331B7EE68FD8:9659365",
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:1",
	}, "\r\n")

	if n, found, err := countInBody(strings.NewReader(body), "1E4C9B93F3F0682250B6CF8331B7EE68FD8"); err != nil || !found || n != 9659365 {
		t.Errorf("matching suffix = (%d, %v, %v), want (9659365, true, nil)", n, found, err)
	}
	// Case-insensitive match (HIBP is uppercase; be forgiving of input).
	if n, found, err := countInBody(strings.NewReader(body), "1e4c9b93f3f0682250b6cf8331b7ee68fd8"); err != nil || !found || n != 9659365 {
		t.Errorf("lowercase suffix should still match, got (%d, %v, %v)", n, found, err)
	}
	// Absent suffix.
	if _, found, err := countInBody(strings.NewReader(body), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"); err != nil || found {
		t.Errorf("absent suffix should not be found, got (found=%v, err=%v)", found, err)
	}
}

// errReader fails partway through, simulating a truncated response body.
type errReader struct {
	data []byte
	read bool
}

func (e *errReader) Read(p []byte) (int, error) {
	if e.read {
		return 0, io.ErrUnexpectedEOF
	}
	e.read = true
	n := copy(p, e.data)
	return n, nil
}

func TestCountInBody_ScanErrorPropagates(t *testing.T) {
	// A body that errors before the suffix is seen must NOT be reported as a
	// clean "not found" — that would tell the user a breached password is safe.
	r := &errReader{data: []byte("0000:0\n")}
	if _, found, err := countInBody(r, "1E4C9B93F3F0682250B6CF8331B7EE68FD8"); err == nil || found {
		t.Errorf("truncated body should surface an error, got (found=%v, err=%v)", found, err)
	}
}

func TestCheck_PwnedAndClean(t *testing.T) {
	// A stub server returns the suffix of SHA-1("password") with a big count.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Add-Padding") != "true" {
			t.Errorf("Add-Padding header not set")
		}
		if !strings.HasSuffix(r.URL.Path, "/5BAA6") {
			t.Errorf("expected prefix path /5BAA6, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("1E4C9B93F3F0682250B6CF8331B7EE68FD8:9659365\r\nAAAA:0\r\n"))
	}))
	defer srv.Close()

	// Point the package at the stub by overriding the base for the test.
	old := apiBase
	apiBase = srv.URL + "/"
	defer func() { apiBase = old }()

	res, err := Check(context.Background(), "password", srv.Client())
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !res.Pwned || res.Count != 9659365 {
		t.Errorf("Check(password) = %+v, want pwned with count 9659365", res)
	}
}
