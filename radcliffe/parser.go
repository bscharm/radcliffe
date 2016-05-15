package radcliffe

import (
	"bytes"
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"sync"

	log "github.com/Sirupsen/logrus"
)

func Parse(wg *sync.WaitGroup, json map[string]interface{}, rootPath string, m chan Metadata) {
	for k, v := range json {
		wg.Add(1)
		p := Pair{Key: k, Value: v, RootPath: rootPath}
		go createMetadata(p, m, wg)
	}
}

func createMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	valueType := reflect.TypeOf(p.Value)
	log.Debugf("Creating metadata for key '%s' of type '%s'", p.Key, valueType)
	switch valueType.String() {
	case BOOL:
		createBoolMetadata(p, m, wg)
		break
	case STRING:
		createStringMetadata(p, m, wg)
		break
	case JSON_NUMBER:
		createNumericMetadata(p, m, wg)
		break
	case MAP_STRING_INTERFACE:
		createMapMetadata(p, m, wg)
		break
	default:
		createUnknownMetadata(p, m, wg)
		break
	}
}

func createMapMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	value := p.Value.(map[string]interface{})
	path := p.RootPath + p.Key
	currentPath := bytes.NewBuffer([]byte(p.RootPath))
	currentPath.WriteString(p.Key)
	currentPath.WriteString(".")
	Parse(wg, value, currentPath.String(), m)
	sendToMetadataChannel(path, OBJECT, "", m, wg)
}

func createBoolMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	path := p.RootPath + p.Key
	sendToMetadataChannel(path, BOOLEAN, "", m, wg)
}

func createStringMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	path := p.RootPath + p.Key
	sendToMetadataChannel(path, STRING, "", m, wg)
}

func createNumericMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	var dataType string
	var format string
	path := p.RootPath + p.Key
	jsonNumber := p.Value.(json.Number)
	value := string(jsonNumber)
	r, _ := regexp.Compile("^[-+]?([0-9]*\\.[0-9]+)$")

	// If we match the regular expression we are dealing with a float, otherwise it is an integer
	if r.MatchString(value) {
		dataType = NUMBER
		f, _ := strconv.ParseFloat(value, 64)
		f = math.Abs(f)
		if f < math.MaxFloat32 {
			format = FLOAT
		} else {
			format = DOUBLE
		}
	} else {
		dataType = INTEGER
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
			format = INT32
		} else {
			format = INT64
		}
	}

	sendToMetadataChannel(path, dataType, format, m, wg)
}

func createUnknownMetadata(p Pair, m chan Metadata, wg *sync.WaitGroup) {
	path := p.RootPath + p.Key
	sendToMetadataChannel(path, UNKNOWN, "", m, wg)
}

func sendToMetadataChannel(path, dataType, format string, m chan Metadata, wg *sync.WaitGroup) {
	md := Metadata{
		Path:      path,
		DataType:  dataType,
		Format:    format,
		WaitGroup: wg,
	}
	m <- md
}
