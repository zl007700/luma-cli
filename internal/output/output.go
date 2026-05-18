package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Envelope is the stable machine-readable output shape for agent callers.
type Envelope struct {
	OK    bool   `json:"ok"`
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// WriteJSON writes a JSON envelope followed by a newline.
func WriteJSON(w io.Writer, env Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
