package alexa

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"strconv"
)

type Lexer interface {
	Parse()
}
type Alexa struct {
	JSON     map[string]interface{}
	Fields   []map[string]interface{}
	RootPath bytes.Buffer
}

func (a *Alexa) Parse() {
	for k, v := range a.JSON {
		valueType := reflect.TypeOf(v)
		switch valueType.String() {
		case "bool":
			a.AddBool(k)
		case "string":
			a.AddString(k)
		case "json.Number":
			a.AddNumber(k, v.(json.Number))
		case "map[string]interface {}":
			a.AddObject(k, v)
		}
	}
}

func (a *Alexa) Path(key string) string {
	if a.RootPath.Len() == 0 {
		return key
	} else {
		currentPath := bytes.NewBuffer(a.RootPath.Bytes())
		currentPath.WriteString(key)
		return currentPath.String()
	}
}

func (a *Alexa) AddObject(k string, v interface{}) {
	currentPath := bytes.NewBuffer(a.RootPath.Bytes())
	currentPath.WriteString(k)
	currentPath.WriteString(".")
	o := Alexa{
		JSON:     v.(map[string]interface{}),
		RootPath: *currentPath,
	}
	o.Parse()
	for _, field := range o.Fields {
		a.Fields = append(a.Fields, field)
	}
}

func (a *Alexa) AddBool(k string) {
	a.Fields = append(a.Fields, map[string]interface{}{
		"type": "boolean",
		"path": a.Path(k),
	})
}

func (a *Alexa) AddString(k string) {
	a.Fields = append(a.Fields, map[string]interface{}{
		"type": "string",
		"path": a.Path(k),
	})
}

func (a *Alexa) AddNumber(k string, v json.Number) {
	var numType string
	var format string
	value := v.String()
	r, _ := regexp.Compile("^[-+]?([0-9]*\\.[0-9]+)$")

	// If we match the regular expression regexp we are dealing with a float
	if r.MatchString(value) {
		numType = "number"
		f, _ := strconv.ParseFloat(value, 64)
		f = math.Abs(f)
		if f < math.MaxFloat32 {
			format = "float"
		} else {
			format = "double"
		}
	} else {
		numType = "integer"
		i, _ := strconv.ParseInt(value, 0, 64)
		if i < 0 {
			i = -i
		}
		if i < math.MaxInt32 {
			format = "int32"
		} else {
			format = "int64"
		}
	}
	a.Fields = append(a.Fields, map[string]interface{}{
		"type":   numType,
		"format": format,
		"path":   a.Path(k),
	})
}
