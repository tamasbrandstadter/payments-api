package web

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"

	"github.com/pkg/errors"
)

type Response struct {
	Results interface{}     `json:"results"`
	Errors  []ResponseError `json:"errors,omitempty"`
}

type ResponseError struct {
	Message string `json:"message"`
}

func (a ResponseError) Error() string {
	return a.Message
}

func Respond(w http.ResponseWriter, r *http.Request, code int, data interface{}, errs ...error) {
	var respErrs []ResponseError

	if len(errs) > 0 {
		for _, err := range errs {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("error while serving request")

			respErrs = append(respErrs, ResponseError{Message: err.Error()})
		}
	}

	resp := Response{
		Results: data,
		Errors:  respErrs,
	}

	writeResponse(w, r, code, &resp)
}

func RespondError(w http.ResponseWriter, r *http.Request, code int, err error) {
	log.WithFields(log.Fields{
		"error": err,
	}).Error("error while serving request")

	if code >= http.StatusInternalServerError && code != http.StatusServiceUnavailable && code != http.StatusNotImplemented {
		code = http.StatusInternalServerError
		err = errors.New(http.StatusText(http.StatusInternalServerError))
	}

	resp := Response{
		Errors: []ResponseError{
			{
				Message: err.Error(),
			},
		},
	}

	writeResponse(w, r, code, &resp)
}

func writeResponse(w http.ResponseWriter, r *http.Request, code int, resp *Response) {
	if code == http.StatusNoContent || resp == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if _, err := w.Write(b); err != nil {
		log.WithError(errors.Wrap(err, "write response body"))
	}
}
