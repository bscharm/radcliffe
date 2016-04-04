package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bscharm/radcliffe/alexa"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"

	log "github.com/Sirupsen/logrus"
)

func main() {
	r := mux.NewRouter()
	registerHandlers(r)
	http.Handle("/", r)
	chain := alice.New(contentType, logging).Then(r)
	startServer(":3000", chain)
}

func registerHandlers(r *mux.Router) {
	r.HandleFunc("/", RadcliffeHandler).Methods("POST")
	r.HandleFunc("/", OptionsHandler).Methods("OPTIONS")
	r.HandleFunc("/", NoMethodHandler)
}

func startServer(p string, h http.Handler) {
	log.Infof("Starting radcliffe on port %s", p)
	http.ListenAndServe(p, h)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "POST")
}

func NoMethodHandler(w http.ResponseWriter, r *http.Request) {
	respondError(
		http.StatusMethodNotAllowed,
		fmt.Sprintf("The %s method is not allowed", r.Method),
		w,
	)
}

func RadcliffeHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	data := make(map[string]interface{})
	if err := dec.Decode(&data); err != nil {
		respondError(
			http.StatusBadRequest,
			"Unable to parse the JSON body",
			w,
		)
		return
	}
	a := alexa.Alexa{JSON: data}
	a.Parse()
	response := a.Fields
	respond(w, response)
}

func respond(w http.ResponseWriter, response []map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(response); err != nil {
		log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func respondError(statusCode int, message string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	log.WithFields(log.Fields{
		"statusCode": statusCode,
	}).Error(message)
	response := map[string]interface{}{
		"error":   true,
		"message": message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithFields(log.Fields{
			"statusCode": 500,
		}).Fatal(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func contentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentTypeUpper := r.Header.Get("Content-Type")
		contentTypeLower := r.Header.Get("content-type")
		noContentType := !(contentTypeUpper == "application/json" || contentTypeLower == "application/json")
		if noContentType {
			respondError(
				http.StatusBadRequest,
				"Content-Type must be set to application/json",
				w,
			)
			return
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}
