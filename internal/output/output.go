// Package output owns the standard response envelope and rendering for all commands.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// State represents the outcome state of a command.
type State string

const (
	StatePlaying     State = "playing"
	StatePaused      State = "paused"
	StateIdle        State = "idle"
	StateNotFound    State = "not_found"
	StateNotLoggedIn State = "not_logged_in"
	StateError       State = "error"
	StateOK          State = "ok"
)

// Envelope is the standard top-level response structure for all commands.
type Envelope struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	State   State  `json:"state"`
	Message string `json:"message"`
	Result  any    `json:"result,omitempty"`
}

// Renderer writes command output to a writer.
type Renderer struct {
	out   io.Writer
	plain bool
}

// NewRenderer creates a Renderer writing to out. If plain is true, it renders
// a single human-readable line instead of JSON.
func NewRenderer(out io.Writer, plain bool) *Renderer {
	return &Renderer{out: out, plain: plain}
}

// Render writes the envelope to the configured writer.
func (r *Renderer) Render(e Envelope) error {
	if r.plain {
		_, err := fmt.Fprintln(r.out, e.Message)
		return err
	}
	enc := json.NewEncoder(r.out)
	enc.SetIndent("", "  ")
	return enc.Encode(e)
}
