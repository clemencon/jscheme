// Package testjson reads the event stream written by `go test -json`.
package testjson

import (
	"bufio"
	"encoding/json"
	"io"
)

// Event is the part of a test2json event the suite's tools use.
type Event struct {
	Action  string // "run", "pass", "fail", "skip", "output", ...
	Package string
	Test    string // empty for package-level events
}

// Read returns the test-level events of package pkg, in order. Lines that are
// not JSON objects (go test can mix in plain build output) are ignored.
func Read(r io.Reader, pkg string) ([]Event, error) {
	var events []Event
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if e.Package == pkg && e.Test != "" {
			events = append(events, e)
		}
	}
	return events, sc.Err()
}
