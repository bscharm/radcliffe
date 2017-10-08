package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"sync"
)

// Parse is responsible for opening the JSON file, and orchestrating the analysis
// of the types present
func Parse(fullPath string) {
	_, filename := filepath.Split(fullPath)
	file, err := validateAndOpenFile(fullPath, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	j, err := decodeJSON(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if reflect.TypeOf(j).String() == ARRAY_MAP_STRING_INTERFACE {
		fmt.Println("json: arrays not currently supported")
		os.Exit(0)
	}

	pairs := make(chan pair)
	go processPairs(j.(map[string]interface{}), pairs, 0, "")

	results := make(chan metadata)
	var wg sync.WaitGroup
	cores := runtime.NumCPU()
	wg.Add(cores)
	go func() {
		wg.Wait()
		close(results)
	}()

	for i := 0; i < cores; i++ {
		go func() {
			defer wg.Done()
			for p := range pairs {
				createMetadata(p, results)
			}
		}()
	}

	outfilename := filename[0:len(filename)-len(filepath.Ext(filename))] + "_out.json"
	outfile, err := os.Create(outfilename)
	defer outfile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var openAPI []metadata
	for r := range results {
		openAPI = append(openAPI, r)
	}

	openAPIJSON, err := json.MarshalIndent(openAPI, "", "\t")
	outfile.Write(openAPIJSON)
	outfile.WriteString("\n")
	fmt.Printf("finished writing file: %s\n", outfilename)
}

const (
	ARRAY                      = "array"
	BOOL                       = "bool"
	BOOLEAN                    = "boolean"
	DOUBLE                     = "double"
	FLOAT                      = "float"
	INTEGER                    = "integer"
	INT32                      = "int32"
	INT64                      = "int64"
	JSON_NUMBER                = "json.Number"
	MAP_STRING_INTERFACE       = "map[string]interface {}"
	ARRAY_MAP_STRING_INTERFACE = "[]map[string]interface {}"
	NUMBER                     = "number"
	OBJECT                     = "object"
	STRING                     = "string"
	UNKNOWN                    = "unknown"
)

type pair struct {
	Key      string
	Value    interface{}
	RootPath string
}

type metadata struct {
	Path         string `json:"path"`
	DataType     string `json:"type"`
	Format       string `json:"format,omitempty"`
	StringType   string `json:"stringType,omitempty"`
	StringFormat string `json:"stringFormat,omitempty"`
}

func validateAndOpenFile(fullPath string, filename string) (*os.File, error) {
	if fullPath == "" {
		return nil, fmt.Errorf("please pass a filename (eg: --file=foo.json)\n")
	}

	extension := filepath.Ext(filename)
	if extension != ".json" {
		return nil, fmt.Errorf("file extension must be .json\n")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v\n", err)
	}

	return file, nil
}

func decodeJSON(file *os.File) (interface{}, error) {
	d := json.NewDecoder(file)
	d.UseNumber()
	jsonMap := make(map[string]interface{})
	if errMap := d.Decode(&jsonMap); errMap != nil {
		file.Seek(0, 0)
		var jsonArray []map[string]interface{}
		if errArray := d.Decode(&jsonArray); errArray != nil {
			return nil, fmt.Errorf("json map error: %v\njson array error: %v\n", errMap, errArray)
		}
		return jsonArray, nil
	}
	return jsonMap, nil
}

func processPairs(json map[string]interface{}, pairs chan pair, depth int, rootPath string) {
	for k, v := range json {
		if reflect.TypeOf(v).String() == MAP_STRING_INTERFACE {
			nested := v.(map[string]interface{})
			processPairs(nested, pairs, depth+1, rootPath+k+".")
		}
		pairs <- pair{k, v, rootPath + k}
	}
	if depth == 0 {
		close(pairs)
	}
}

func createMetadata(p pair, m chan metadata) {
	valueType := reflect.TypeOf(p.Value)
	md := metadata{
		Path: p.RootPath,
	}

	switch valueType.String() {
	case BOOL:
		md.DataType = BOOLEAN
		break
	case STRING:
		md.DataType = STRING
		break
	case JSON_NUMBER:
		dataType, format := getNumericMetadata(p, m)
		md.DataType = dataType
		md.Format = format
		break
	case MAP_STRING_INTERFACE:
		md.DataType = OBJECT
		break
	default:
		md.DataType = UNKNOWN
		break
	}
	m <- md
}

func getNumericMetadata(p pair, m chan metadata) (string, string) {
	var dataType string
	var format string

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

	return dataType, format
}
