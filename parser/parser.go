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

	if reflect.TypeOf(j).String() == JSONArray {
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

// Constants representing various types
const (
	Array      = "array"
	Bool       = "bool"
	Boolean    = "boolean"
	Date       = "date"
	DateTime   = "date-time"
	Double     = "double"
	Float      = "float"
	Integer    = "integer"
	Int32      = "int32"
	Int64      = "int64"
	JSONNumber = "json.Number"
	JSONMap    = "map[string]interface {}"
	JSONArray  = "[]map[string]interface {}"
	Number     = "number"
	Object     = "object"
	String     = "string"
	Unknown    = "unknown"
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
		return nil, fmt.Errorf("please pass a filename (eg: --file=foo.json)")
	}

	extension := filepath.Ext(filename)
	if extension != ".json" {
		return nil, fmt.Errorf("file extension must be .json")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
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
			return nil, fmt.Errorf("json map error: %v\njson array error: %v", errMap, errArray)
		}
		return jsonArray, nil
	}
	return jsonMap, nil
}

func processPairs(json map[string]interface{}, pairs chan pair, depth int, rootPath string) {
	for k, v := range json {
		if reflect.TypeOf(v).String() == JSONMap {
			nested := v.(map[string]interface{})
			processPairs(nested, pairs, depth+1, rootPath+k+".")
		}
		pairs <- pair{k, v, rootPath + k}
	}
	if depth == 0 {
		close(pairs)
	}
}

func createMetadata(p pair, m chan<- metadata) {
	valueType := reflect.TypeOf(p.Value)
	md := metadata{
		Path: p.RootPath,
	}

	switch valueType.String() {
	case Bool:
		md.DataType = Boolean
		break
	case String:
		format := getStringMetadata(p)
		md.DataType = String
		if format != "" {
			md.Format = format
		}
		break
	case JSONNumber:
		dataType, format := getNumericMetadata(p)
		md.DataType = dataType
		md.Format = format
		break
	case JSONMap:
		md.DataType = Object
		break
	default:
		md.DataType = Unknown
		break
	}
	m <- md
}

func getStringMetadata(p pair) (format string) {
	fullDate, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}")
	dateTimeTimeSecfracZulu, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{1,4}Z")
	dateTimeTimeSecfracOffset, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{1,4}(\\-|\\+)\\d{2}:\\d{2}")
	dateTimeOffset, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\-|\\+)\\d{2}:\\d{2}")
	dateTimeZulu, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z")

	value := p.Value.(string)
	if dateTimeTimeSecfracOffset.MatchString(value) ||
		dateTimeTimeSecfracZulu.MatchString(value) ||
		dateTimeOffset.MatchString(value) ||
		dateTimeZulu.MatchString(value) {
		return DateTime
	} else if fullDate.MatchString(value) {
		return Date
	}

	return format
}

func getNumericMetadata(p pair) (string, string) {
	var dataType string
	var format string

	jsonNumber := p.Value.(json.Number)
	value := string(jsonNumber)
	r, _ := regexp.Compile("^[-+]?([0-9]*\\.[0-9]+)$")

	// If we match the regular expression we are dealing with a float, otherwise it is an integer
	if r.MatchString(value) {
		dataType = Number
		f, _ := strconv.ParseFloat(value, 64)
		f = math.Abs(f)
		if f < math.MaxFloat32 {
			format = Float
		} else {
			format = Double
		}
	} else {
		dataType = Integer
		bi := big.NewInt(0)
		_, ok := bi.SetString(value, 10)
		if ok != true {
		}
		i := bi.Int64()
		if i < 0 {
			i = -i
		}
		if i < math.MaxInt32 {
			format = Int32
		} else {
			format = Int64
		}
	}

	return dataType, format
}
