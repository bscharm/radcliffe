package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"sync"
)

func main() {
	filename := flag.String("file", "", "Filename to parse")
	flag.Parse()

	if *filename == "" {
		fmt.Fprint(os.Stderr, "please pass a filename (eg: --file=foo.json)\n")
		os.Exit(1)
	}

	extension := filepath.Ext(*filename)
	if extension != ".json" {
		fmt.Fprintf(os.Stderr, "file extension must be .json\n")
		os.Exit(1)
	}

	file, err := os.Open(*filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file: %v\n", err)
		os.Exit(1)
	}

	d := json.NewDecoder(file)
	d.UseNumber()
	jsonMap := make(map[string]interface{})
	if err := d.Decode(&jsonMap); err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding json: %v\n", err)
		os.Exit(1)
	}

	// nodes := countNodes(json)
	pairs := make(chan Pair)
	go processPairs(jsonMap, pairs, 0, "")

	results := make(chan Metadata)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		wg.Wait()
		close(results)
	}()

	for i := 0; i < 4; i++ {
		go func() {
			defer wg.Done()
			for p := range pairs {
				createMetadata(p, results)
			}
		}()
	}

	outfile, err := os.Create("out_" + *filename)
	defer outfile.Close()
	for r := range results {
		j, _ := json.Marshal(r)
		outfile.Write(j)
		outfile.WriteString("\n")
	}
}

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

// Pair represents a JSON key and value, as well as the root path for that key
type Pair struct {
	Key      string
	Value    interface{}
	RootPath string
}

// Metadata is what we return for each key/value pair in a JSON payload. Format, String Format and
// String Type are optional depending on the type of the value. We pass through the WaitGroup so our
// response waits until all key/value pairs in the JSON payload are done.
type Metadata struct {
	Path         string `json:"path"`
	DataType     string `json:"type"`
	Format       string `json:"format,omitempty"`
	StringType   string `json:"stringType,omitempty"`
	StringFormat string `json:"stringFormat,omitempty"`
}

func processPairs(json map[string]interface{}, pairs chan Pair, depth int, rootPath string) {
	for k, v := range json {
		if reflect.TypeOf(v).String() == MAP_STRING_INTERFACE {
			nested := v.(map[string]interface{})
			processPairs(nested, pairs, depth+1, rootPath+k+".")
		}
		pairs <- Pair{k, v, rootPath + k}
	}
	if depth == 0 {
		close(pairs)
	}
}

func createMetadata(p Pair, m chan Metadata) {
	valueType := reflect.TypeOf(p.Value)
	switch valueType.String() {
	case BOOL:
		createBoolMetadata(p, m)
		break
	case STRING:
		createStringMetadata(p, m)
		break
	case JSON_NUMBER:
		createNumericMetadata(p, m)
		break
	case MAP_STRING_INTERFACE:
		createMapMetadata(p, m)
		break
	default:
		createUnknownMetadata(p, m)
		break
	}
}

func createMapMetadata(p Pair, m chan Metadata) {
	path := p.RootPath
	sendToMetadataChannel(path, OBJECT, "", m)
}

func createBoolMetadata(p Pair, m chan Metadata) {
	path := p.RootPath
	sendToMetadataChannel(path, BOOLEAN, "", m)
}

func createStringMetadata(p Pair, m chan Metadata) {
	path := p.RootPath
	sendToMetadataChannel(path, STRING, "", m)
}

func createNumericMetadata(p Pair, m chan Metadata) {
	var dataType string
	var format string
	path := p.RootPath
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

	sendToMetadataChannel(path, dataType, format, m)
}

func createUnknownMetadata(p Pair, m chan Metadata) {
	path := p.RootPath
	sendToMetadataChannel(path, UNKNOWN, "", m)
}

func sendToMetadataChannel(path, dataType, format string, m chan Metadata) {
	md := Metadata{
		Path:     path,
		DataType: dataType,
		Format:   format,
	}
	m <- md
}
