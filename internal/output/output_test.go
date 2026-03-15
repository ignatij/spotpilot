package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ignatij/spotpilot/internal/output"
)

func TestRenderer_JSON(t *testing.T) {
	env := output.Envelope{
		OK:      true,
		Command: "play",
		State:   output.StatePlaying,
		Message: "Playing Master of Puppets",
	}

	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	if err := r.Render(env); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var got output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if !got.OK {
		t.Error("expected ok=true")
	}
	if got.Command != "play" {
		t.Errorf("expected command=play, got %q", got.Command)
	}
	if got.State != output.StatePlaying {
		t.Errorf("expected state=playing, got %q", got.State)
	}
}

func TestRenderer_Plain(t *testing.T) {
	env := output.Envelope{
		OK:      true,
		Command: "play",
		State:   output.StatePlaying,
		Message: "Playing: Master of Puppets — Metallica",
	}

	var buf bytes.Buffer
	r := output.NewRenderer(&buf, true)
	if err := r.Render(env); err != nil {
		t.Fatalf("Render: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if line != "Playing: Master of Puppets — Metallica" {
		t.Errorf("unexpected plain output: %q", line)
	}
}

func TestErrorEnvelope(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantState output.State
		wantExit  int
	}{
		{
			name:      "not found",
			err:       &output.AppError{Cat: output.CategoryNotFound, Message: "not found"},
			wantState: output.StateNotFound,
			wantExit:  4,
		},
		{
			name:      "auth error",
			err:       &output.AppError{Cat: output.CategoryAuth, Message: "not logged in"},
			wantState: output.StateNotLoggedIn,
			wantExit:  3,
		},
		{
			name:      "validation error",
			err:       &output.AppError{Cat: output.CategoryValidation, Message: "invalid input"},
			wantState: output.StateError,
			wantExit:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := output.ErrorEnvelope("test", tc.err)
			if env.OK {
				t.Error("expected ok=false")
			}
			if env.State != tc.wantState {
				t.Errorf("expected state=%q, got %q", tc.wantState, env.State)
			}
			if got := output.ExitCode(tc.err); got != tc.wantExit {
				t.Errorf("expected exit=%d, got %d", tc.wantExit, got)
			}
		})
	}
}
