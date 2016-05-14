package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRadcliffeHandler(t *testing.T) {

	test := []byte(`{"foo":1,"bar":"7.7"}`)
	reader := bytes.NewReader(test)

	req, err := http.NewRequest("POST", "/", reader)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RadcliffeHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	t.Logf("Body was %s", rr.Body.String())

}
