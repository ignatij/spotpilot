package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type debugLogger struct {
	w io.Writer
}

func (d debugLogger) enabled() bool {
	return d.w != nil
}

func (d debugLogger) printf(format string, args ...any) {
	if !d.enabled() {
		return
	}
	_, _ = fmt.Fprintf(d.w, "spotpilot debug: "+format+"\n", args...)
}

func (d debugLogger) dumpJSON(label string, value any) {
	if !d.enabled() {
		return
	}
	d.printf("%s", label)
	encoder := json.NewEncoder(d.w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
	_, _ = io.WriteString(d.w, strings.Repeat("-", 40)+"\n")
}
