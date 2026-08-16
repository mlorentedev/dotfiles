package secrets

import (
	"fmt"
	"io"
	"net/http"
)

// ProbeResult is the transport-level truth about one bw serve response: what the daemon
// answered, not what it meant. Body is carried in memory for a caller that must INSPECT
// a failure rather than consume a value; nothing in this package prints it.
type ProbeResult struct {
	Status      int    // HTTP status as received; 0 when the request never completed
	ContentType string // raw Content-Type header
	Size        int    // len(body) as received
	Body        []byte // NEVER logged, printed, or wrapped into an error by this package
}

// Probe issues method to path and returns the raw transport facts WITHOUT decoding an
// envelope or interpreting success.
//
// Deliberately NOT named call-anything, and deliberately not a flag on call(): call() is
// the safe default that never touches body bytes, and this is the one path that does.
// Keeping them separately named means a future refactor cannot route an ordinary read
// through the body-carrying path by accident — the failure mode being designed against
// is a credential in a log. Three separate credential exposures in this repo in one day
// came from ad-hoc inspection of exactly these responses, which is why the safe
// instrument exists at all (CLI-038, #1012).
//
// Contract:
//
//   - It uses the SAME baseURL and http.Client as call(), so it reproduces the
//     connection policy of the path it exists to diagnose. A hand-rolled client would
//     not — which is precisely why curl could not characterise #988.
//   - A transport failure (nothing listening, timeout) returns ErrBWServeUnreachable
//     wrapped, with Status 0, matching call()'s contract so poll-loop semantics are
//     unaffected.
//   - A non-2xx is NOT an error. It is the successful observation of a failure, which is
//     the entire purpose; the caller decides severity.
//   - No error returned here ever contains body bytes.
//
// Callers MUST treat Body as secret-bearing regardless of status: a 200 from
// /object/item/<id> IS the credential.
func (c BWServeClient) Probe(method, path string) (ProbeResult, error) {
	req, err := http.NewRequest(method, c.baseURL()+path, nil) //nolint:noctx // bounded by the client's own Timeout
	if err != nil {
		return ProbeResult{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("%w: %s %s: %v", ErrBWServeUnreachable, method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	res := ProbeResult{
		Status:      resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        len(body),
		Body:        body,
	}
	if readErr != nil {
		// Report the failure to read WITHOUT the partial body: a truncated read of a
		// 200 is still credential material.
		//
		// Body is cleared rather than merely kept out of the error string. The
		// comment above said "without the partial body" while the struct still
		// carried it, which is the exact defect class this whole ticket is about:
		// an assertion the code does not implement. Size is kept -- it is metadata,
		// and how much arrived before the read died is a real diagnostic fact.
		res.Body = nil
		return res, fmt.Errorf("%s %s: reading bw serve response (HTTP %d): %w", method, path, resp.StatusCode, readErr)
	}
	return res, nil
}
