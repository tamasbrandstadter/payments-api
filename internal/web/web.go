package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func RespondError(w http.ResponseWriter, code int, message string) {
	log.Error("error while serving request: ", message)

	if code >= http.StatusInternalServerError && code != http.StatusServiceUnavailable && code != http.StatusNotImplemented {
		code = http.StatusInternalServerError
		message = http.StatusText(http.StatusInternalServerError)
	}

	Respond(w, code, map[string]string{"error": message})
}

func Respond(w http.ResponseWriter, code int, payload interface{}) {
	if code == http.StatusNoContent || payload == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to marshal response: %s", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		log.Error("could not write response: ", err)
	}

}
