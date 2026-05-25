package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Envelope is the stable machine-readable output shape for agent callers.
type Envelope struct {
	OK     bool   `json:"ok"`
	Code   string `json:"code,omitempty"`
	Error  string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
	Notice any    `json:"_notice,omitempty"`
}

// PendingNotice, when set, returns notices to inject as
// the "_notice" field in every JSON envelope. Set by main.go
// after skills drift and update checks.
var PendingNotice func() any

// GetNotice returns the current pending notice, or nil.
func GetNotice() any {
	if PendingNotice == nil {
		return nil
	}
	return PendingNotice()
}

// WriteJSON writes a JSON envelope followed by a newline.
// If PendingNotice is set, the notice is injected into the envelope.
func WriteJSON(w io.Writer, env Envelope) error {
	if env.Notice == nil && PendingNotice != nil {
		if notice := PendingNotice(); notice != nil {
			env.Notice = notice
		}
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
