package blob

import "time"

// retry runs op, repeating transient failures (Windows sharing/lock/access-denied while another
// handle is open, antivirus scans) with exponential backoff for up to ~2 s. After that the error is
// returned and the caller leaves the repair to the consistency check: the database is the truth.
func retry(op func() error) error {
	delay := 10 * time.Millisecond
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := op()
		if err == nil || !transient(err) || !time.Now().Add(delay).Before(deadline) {
			return err
		}
		time.Sleep(delay)
		if delay < 320*time.Millisecond {
			delay *= 2
		}
	}
}
