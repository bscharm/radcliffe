package radcliffe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

var (
	// PORT is the port to run the server on
	PORT  string
	DEBUG bool
)

const (
	ARRAY                = "array"
	BOOL                 = "bool"
	BOOLEAN              = "boolean"
	DOUBLE               = "double"
	FLOAT                = "float"
	INTEGER              = "integer"
	INT32                = "int32"
	INT64                = "int64"
	JSON_NUMBER          = "json.Number"
	MAP_STRING_INTERFACE = "map[string]interface {}"
	NUMBER               = "number"
	OBJECT               = "object"
	STRING               = "string"
	UNKNOWN              = "unknown"
)

type Pair struct {
	Key      string
	Value    interface{}
	RootPath string
}

type Metadata struct {
	Path         string          `json:"path"`
	DataType     string          `json:"type"`
	Format       string          `json:"format,omitempty"`
	StringType   string          `json:"stringType,omitempty"`
	StringFormat string          `json:"stringFormat,omitempty"`
	WaitGroup    *sync.WaitGroup `json:"-"`
}

func Start() {
	if DEBUG {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	r := mux.NewRouter()
	registerHandlers(r)
	http.Handle("/", r)
	chain := alice.New(contentType, logging).Then(r)
	startServer(fmt.Sprintf(":%s", PORT), chain)
}

func startServer(p string, h http.Handler) {
	log.Infof("Starting radcliffe on port %s", p)
	log.Fatal(http.ListenAndServe(p, h))
}

func registerHandlers(r *mux.Router) {
	r.HandleFunc("/", RadcliffeHandler).Methods("POST")
	r.HandleFunc("/", OptionsHandler).Methods("OPTIONS")
	r.HandleFunc("/", NoMethodHandler)
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
	var wg sync.WaitGroup
	var response []Metadata
	m := make(chan Metadata)
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

	go buildResponse(&response, m)
	Parse(&wg, data, "", m)
	wg.Wait()
	respond(w, response)
}

func buildResponse(response *[]Metadata, m chan Metadata) {
	for md := range m {
		*response = append(*response, md)
		md.WaitGroup.Done()
	}
}

func respond(w http.ResponseWriter, response []Metadata) {
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
