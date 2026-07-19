// Package breach checks a password against the Have I Been Pwned (HIBP)
// Pwned Passwords corpus using the k-anonymity range API. The plaintext
// password never leaves the machine: only the first 5 hex characters of its
// SHA-1 hash are sent, and the matching suffix is compared locally.
package breach

import (
	"bufio"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// apiBase is the HIBP range endpoint. A prefix of 5 hex chars is appended.
// It is a var (not a const) so tests can point Check at a stub server.
var apiBase = "https://api.pwnedpasswords.com/range/"

// defaultClient is reused across calls so the HTTP transport can pool
// connections; constructing a fresh http.Client per call defeats keep-alive
// and leaks idle dialer resources.
var defaultClient = &http.Client{Timeout: 10 * time.Second}

// hashParts returns the uppercase SHA-1 of the password split into the 5-char
// range prefix (sent to the API) and the remaining suffix (matched locally).
// HIBP stores and returns hashes in uppercase hex.
func hashParts(password string) (prefix, suffix string) {
	sum := sha1.Sum([]byte(password))
	full := fmt.Sprintf("%X", sum[:])
	return full[:5], full[5:]
}

// countInBody scans an HIBP range response for the given suffix and returns
// its breach count. The body is newline-separated "SUFFIX:COUNT" lines; the
// padding entries HIBP injects (count 0) are handled naturally since a real
// match carries a positive count. found reports whether the suffix appeared.
//
// A read/scan error (e.g. a truncated response) is returned rather than
// swallowed: treating a partial body as "not found" would tell the user a
// compromised password is clean, so the caller must be able to distinguish
// "confirmed absent" from "could not finish reading".
func countInBody(body io.Reader, suffix string) (count int, found bool, err error) {
	sc := bufio.NewScanner(body)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		suf, cnt, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(suf, suffix) {
			continue
		}
		n, aerr := strconv.Atoi(strings.TrimSpace(cnt))
		if aerr != nil {
			return 0, false, aerr
		}
		return n, true, nil
	}
	if serr := sc.Err(); serr != nil {
		return 0, false, serr
	}
	return 0, false, nil
}

// Result is the outcome of a breach check.
type Result struct {
	// Count is how many times the password appears in the corpus. 0 means it
	// was not found (Pwned is false).
	Count int
	// Pwned is true when the password appears at least once.
	Pwned bool
}

// Check queries HIBP for the password via the k-anonymity range API. Pass nil
// for client to use a default 10s-timeout client. Only the 5-char SHA-1 prefix
// is transmitted; the suffix match is done locally, so the password is never
// sent in full.
func Check(ctx context.Context, password string, client *http.Client) (Result, error) {
	if client == nil {
		client = defaultClient
	}
	prefix, suffix := hashParts(password)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+prefix, nil)
	if err != nil {
		return Result{}, err
	}
	// Add-Padding asks HIBP to pad the response with decoy hashes so the
	// returned size does not leak how many real suffixes share the prefix.
	req.Header.Set("Add-Padding", "true")
	req.Header.Set("User-Agent", "brtc-password-cost")

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	// Drain any unread body before closing so the transport can reuse the
	// connection: countInBody returns early on a match, leaving bytes behind.
	// Cap the drain with a LimitReader so a malicious/huge response cannot pin
	// the goroutine reading an unbounded stream. A real range response is only
	// a few tens of KB; 1 MiB is comfortable headroom.
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("hibp: unexpected status %d", resp.StatusCode)
	}

	count, found, err := countInBody(resp.Body, suffix)
	if err != nil {
		return Result{}, err
	}
	return Result{Count: count, Pwned: found && count > 0}, nil
}
