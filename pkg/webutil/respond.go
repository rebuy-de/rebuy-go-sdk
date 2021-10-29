package webutil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RespondJSON sets the proper content type and sends the given data as JSON to
// the client.
func RespondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")

	err := enc.Encode(data)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// RespondError sends an error ID to the client and logs the error, if it is
// not nil. It returns true, if the error was not nil. This makes it possible
// to do condensed error checking:
//
//     err := DoSomething()
//     if webutil.RespondError(w, err) {
//         return
//     }
func RespondError(w http.ResponseWriter, err error) bool {
	if err != nil {
		id := uuid.New()

		logrus.
			WithError(err).
			WithField("uuid", id.String()).
			Info("failed to handle request")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "ERROR: %s", id.String())
		return true
	}

	return false
}
