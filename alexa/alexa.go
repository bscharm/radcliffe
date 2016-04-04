package alexa

import (
	"bytes"
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

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
			a.addBool(k)
		case "string":
			a.addString(k)
		case "json.Number":
			a.addNumber(k, v.(json.Number))
		case "map[string]interface {}":
			a.addObject(k, v.(map[string]interface{}))
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

func (a *Alexa) addObject(k string, v map[string]interface{}) {
	currentPath := bytes.NewBuffer(a.RootPath.Bytes())
	currentPath.WriteString(k)
	currentPath.WriteString(".")
	innerObject := Alexa{
		JSON:     v,
		RootPath: *currentPath,
	}
	innerObject.Parse()
	for _, field := range innerObject.Fields {
		a.Fields = append(a.Fields, field)
	}
}

func (a *Alexa) addBool(k string) {
	a.Fields = append(a.Fields, map[string]interface{}{
		"type": "boolean",
		"path": a.Path(k),
	})
}

func (a *Alexa) addString(k string) {
	a.Fields = append(a.Fields, map[string]interface{}{
		"type": "string",
		"path": a.Path(k),
	})
}

func (a *Alexa) addNumber(k string, v json.Number) {
	var numType string
	var format string
	value := v.String()
	r, _ := regexp.Compile("^[-+]?([0-9]*\\.[0-9]+)$")

	// If we match the regular expression we are dealing with a float, otherwise it is an integer
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
		bi := big.NewInt(0)
		_, ok := bi.SetString(value, 10)
		if ok != true {
			log.WithFields(log.Fields{
				"message": "Unable to parse the Integer string",
			}).Error("error parsing integer value")
		}
		i := bi.Int64()
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
