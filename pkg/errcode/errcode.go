// Package errcode defines the structured error type that crosses the Wails
// bridge. A code doubles as the vue-i18n key; optional params feed i18n
// interpolation. See BACKEND_I18N_HANDOFF.md §3.1.
package errcode

import "encoding/json"

// Error carries a stable code (doubles as vue-i18n key) plus optional i18n
// interpolation params. Immutable: unexported fields, build via New/Newf only.
type Error struct {
	code   string
	params map[string]any
}

func New(code string) *Error { return &Error{code: code} }

func Newf(code string, params map[string]any) *Error {
	p := make(map[string]any, len(params))
	for k, v := range params {
		p[k] = v
	}
	return &Error{code: code, params: p}
}

// Error returns the bare code — readable in tests and devtools.
// Structured data crosses the bridge via MarshalJSON + ErrorFormatter, not here.
func (e *Error) Error() string { return e.code }

// MarshalJSON serializes the structured payload for the Wails bridge.
// Must NEVER return an error — dispatcher calls log.Fatal on marshal failure.
// Current impl is infallible (json.Marshal of a {code, params} struct).
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code   string         `json:"code"`
		Params map[string]any `json:"params,omitempty"`
	}{e.code, e.params})
}