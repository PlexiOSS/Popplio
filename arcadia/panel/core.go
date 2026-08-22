package panel

import (
	"encoding/json"
	"io"
	"net/http"

	"popplio/state"

	"go.uber.org/zap"
)

type Error struct {
	Status  int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func newError(err error) Error {
	return Error{Status: http.StatusInternalServerError, Message: err.Error()}
}

func errStatus(status int, message string) Error {
	return Error{Status: status, Message: message}
}

type response struct {
	status int
	json   any
	text   *string
	stream io.ReadCloser
	noBody bool
}

func writeJSON(status int, v any) response {
	return response{status: status, json: v}
}

func writeText(status int, s string) response {
	return response{status: status, text: &s}
}

func writeNoContent() response {
	return response{status: http.StatusNoContent, noBody: true}
}

func (r response) write(w http.ResponseWriter) {
	switch {
	case r.stream != nil:
		defer r.stream.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(r.status)

		if _, err := io.Copy(w, r.stream); err != nil {
			state.Logger.Error("panel: failed streaming response body", zap.Error(err))
		}
	case r.json != nil:
		body, err := json.Marshal(r.json)

		if err != nil {
			state.Logger.Error("panel: failed marshalling response", zap.Error(err))
			writeText(http.StatusInternalServerError, err.Error()).write(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(r.status)
		w.Write(body)
	case r.text != nil:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(r.status)
		w.Write([]byte(*r.text))
	default:
		w.WriteHeader(r.status)
	}
}
