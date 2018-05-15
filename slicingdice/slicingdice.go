// A library that provides a Go client to Slicing Dice API
package slicingdice

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"reflect"
	"fmt"
	"strings"
	"log"
)

var sd_base = os.Getenv("SD_API_ADDRESS")

// All these constants representing all endpoints available in SlicingDice API.
const (
	RESULT             = "/data_extraction/result/"
	SCORE              = "/data_extraction/score/"
	INSERT             = "/insert/"
	COLUMN             = "/column/"
	DATABASE           = "/database/"
	TOP_VALUES         = "/query/top_values/"
	EXISTS_ENTITY      = "/query/exists/entity/"
	COUNT_ENTITY       = "/query/count/entity/"
	COUNT_ENTITY_TOTAL = "/query/count/entity/total/"
	COUNT_EVENT        = "/query/count/event/"
	AGGREGATION        = "/query/aggregation/"
	SAVED              = "/query/saved/"
	SQL				   = "/sql/"
)

/* APIKey is used to access the keys that we insert in the SlicingDice API.
There is only one rule: if you put the master key, you do not put the other,
because already by default the client uses for everything. Otherwise,
use a key to writing, if you want to write (insert data, create columns)
and use a key to reading (make queries).
*/
type APIKey struct {
	WriteKey  string
	ReadKey   string
	MasterKey string
	CustomKey string
}

// SlicingDice is the main structure of slicingdice. Through it, we will make queries,
// we will create columns, we'll take databases, etc.
type SlicingDice struct {
	key     map[string]string
	timeout int
	Test    bool
}

// stringInSlice checks if a array has a item.
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// hasValidSavedQuery checks whether the query passed by user is valid. It
// validates especially if the query type is valid.
func hasValidSavedQuery(query interface{}) error {
	queryConverted := query.(map[string]interface{})
	listQueryTypes := []string{"count/entity", "count/event", "count/entity/total", "aggregation", "top_values"}
	if !stringInSlice(queryConverted["type"].(string), listQueryTypes) {
		return errors.New("Saved Query Validator: this dictionary don't have query type valid.")
	}
	return nil
}

// hasValidCountQuery checks whether the count query passed by user is valid. It
// validates especially if the query len is less than 10.
func hasValidCountQuery(query interface{}) error {
	switch query.(type) {
	case []interface{}:
		querySize := len(query.([]interface{}))
		if querySize > 10 {
			return errors.New("Count Query Validator: the query count entity has a limit of 10 queries by request.")
		}
	}

	return nil
}

// hasValidTopValuesQuery checks whether the top values query passed by user is valid. It
// validates especially if the query len is less than 5 and column len is less than 6.
func hasValidTopValuesQuery(query interface{}) error {
	queryConverted := query.(map[string]interface{})
	// check query limit
	if len(queryConverted) > 5 {
		return errors.New("Top Values Validator: the top values query has a limit of 5 queries by request.")
	}
	// check column limit
	for _, value := range queryConverted {
		if len(value.(map[string]interface{})) > 6 {
			return errors.New("Top Values Validator: the query exceeds the limit of columns per query in request")
		}
	}
	return nil
}

// hasValidDataExtractionQuery checks whether the data extraction(result and score)
// query passed by user is valid. It validates especially if the 'limit' key
// has a len less than 100 and if has a valid column.
func hasValidDataExtractionQuery(query interface{}) error {
	queryConverted := query.(map[string]interface{})
	if val, ok := queryConverted["columns"]; ok {
		columns := reflect.ValueOf(val)
		if columns.Len() > 10 {
			return errors.New("Data Extraction Validator: The key 'columns' in data extraction result must have up to 10 columns.")
		}
	}
	return nil
}

// hasValidColumn checks whether the new column is valid. Checks type, name,
// description; enumerate, decimal-place and string types.
func hasValidColumn(query interface{}) error {
	if reflect.ValueOf(query).Kind() == reflect.Slice {
		columnData := query.([]interface{})
		for _, column := range columnData {
			column := column.(map[string]interface{})
	        validateColumn(column)
	    }
	} else {
		query := query.(map[string]interface{})
		validateColumn(query)
	}
	return nil
}

func validateColumn(query map[string]interface{}) error {
	validTypeColumns := []string{
		"unique-id", "boolean", "string", "integer", "decimal",
		"enumerated", "date", "integer-time-series",
		"decimal-time-series", "string-time-series", "datetime",
	}
	// validate name
	if _, ok := query["name"]; !ok {
		return errors.New("Column Validator: the column should have a name.")
	}
	name := query["name"]
	if len(name.(string)) > 80 {
		return errors.New("Column Validator: the column's name have a very big name.(Max: 80 chars)")
	}
	// validate description
	if _, ok := query["description"]; ok {
		description := query["description"]
		if len(description.(string)) > 300 {
			return errors.New("Column Validator: the column's description have a very big name.(Max: 300chars)")
		}
	}
	// validate type column
	if _, ok := query["type"]; !ok {
		return errors.New("Column Validator: the column should have a type.")
	}
	typeColumn := query["type"].(string)
	if !stringInSlice(typeColumn, validTypeColumns) {
		return errors.New("Column Validator: this column have a invalid type.")
	}
	// validate decimal place key
	if _, ok := query["decimal-place"]; ok {
		decimalTypes := []string{"decimal", "decimal-time-series"}
		if !stringInSlice(query["type"].(string), decimalTypes) {
			return errors.New("Column Validator: this column is only accepted on type 'decimal' or 'decimal-time-series'.")
		}
	}
	// validate string column type
	if query["type"] == "string" {
		if _, ok := query["cardinality"]; !ok {
			return errors.New("Column Validator: the column with type string should have 'cardinality' key.")
		}
		cardinalityTypes := []string{"high", "low"}
		if !stringInSlice(query["cardinality"].(string), cardinalityTypes) {
			return errors.New("Column Validator: the column 'cardinality' has invalid value.")
		}
	}
	// validate enumerated column
	if query["type"] == "enumerated" {
		if _, ok := query["range"]; !ok {
			return errors.New("Column Validator: the 'enumerate' type needs of the 'range' parameter.")
		}
	}
	return nil
}

// New returns a new SlicingDice object.
func New(key *APIKey, timeout int) *SlicingDice {
	SlicingDice := new(SlicingDice)
	if len(key.MasterKey) != 0 {
		SlicingDice.key = map[string]string{
			"masterKey": key.MasterKey,
		}
	} else if len(key.CustomKey) != 0 {
		SlicingDice.key = map[string]string{
			"customKey": key.CustomKey,
		}
	} else {
		SlicingDice.key = map[string]string{
			"readKey":  key.ReadKey,
			"writeKey": key.WriteKey,
		}
	}
	SlicingDice.timeout = timeout
	return SlicingDice
}

func (s *SlicingDice) getKeyLevel(keys map[string]string) int {
	if len(keys["masterKey"]) != 0 || len(keys["customKey"]) != 0 {
		return 2
	} else if len(keys["writeKey"]) != 0 {
		return 1
	} else if len(keys["readKey"]) != 0 {
		return 0
	}
	return -1
}

// getKey returns of smart way a key to be used in the request SlicingDice API.
func (s *SlicingDice) getKey(keys map[string]string, endpointKeyLevel int) (string, error) {
	currentKeyLevel := s.getKeyLevel(keys)
	if currentKeyLevel == 2 {
		if len(keys["masterKey"]) != 0 {
			return keys["masterKey"], nil
		} else if len(keys["customKey"]) != 0 {
			return keys["customKey"], nil
		}
	} else if currentKeyLevel != endpointKeyLevel {
		return "", errors.New("API key: This key don't have permission to peform this operation")
	} else {
		if len(keys["writeKey"]) != 0 {
			return keys["writeKey"], nil
		}
		if len(keys["readKey"]) != 0 {
			return keys["readKey"], nil
		}
	}
	return "", nil
}

// getFullUrl checks if enviroment has the SD_API_ADDRESS variable. If don't has
// him define the url base how 'https://api.slicingdice.com/v1'.
func (s *SlicingDice) getFullUrl(path string) string {
	if len(sd_base) != 0 {
		return sd_base + path
	} 
	return "https://api.slicingdice.com/v1" + path
}

func (s *SlicingDice) makeRequest(url string, method string, endpointKeyLevel int, query interface{}) (map[string]interface{}, error) {
	return s.makeRequestSQL(url, method, endpointKeyLevel, query, false)
}

// makeRequest checks request method, convert the query passed for use to JSON
// and executes the request.
func (s *SlicingDice) makeRequestSQL(url string, method string, endpointKeyLevel int, query interface{}, sql bool) (map[string]interface{}, error) {
	methodsAllowed := []string{"GET", "POST", "PUT", "DELETE"}
	if !stringInSlice(method, methodsAllowed) {
		return nil, errors.New("request: this is a invalid method to make request.")
	}
	key, err := s.getKey(s.key, endpointKeyLevel)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(time.Duration(s.timeout) * time.Second)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}


	var request *http.Request
	var err_request error
	var contentType string
	if sql {
		contentType = "application/sql"
		queryData := []byte(query.(string))
		request, err_request = http.NewRequest(method, url, bytes.NewBuffer(queryData))
	} else {
		contentType = "application/json"
		queryData := new(bytes.Buffer)
		json.NewEncoder(queryData).Encode(query)
		request, err_request = http.NewRequest(method, url, queryData)
	}

	if err_request != nil {
		return nil, err_request
	}

	request.Header.Add("Authorization", key)
	request.Header.Add("Content-Type", contentType)
	res, err := client.Do(request)
	return s.handlerResponse(res, err)
}

type SDError struct {
	message string
	moreInfo interface{}
	code int
}

func (e *SDError) Error() string {
	return fmt.Sprintf("Error Code: %d, Message: %s, More Info: %s", e.code, e.message, e.moreInfo)
}


// handlerResponse search errors in the response of the request, both the
// status code as in the API JSON response.
func (s *SlicingDice) handlerResponse(res *http.Response, err error) (map[string]interface{}, error) {
	if err != nil {
		return nil, err
	}
	// check api response json
	defer res.Body.Close()
	contents, _ := ioutil.ReadAll(res.Body)
	result := string(contents)
	decodeResponse, err := s.DecodeJSON(result)
	if err != nil {
		return nil, &SDError{"Response parsing error", result, res.StatusCode}
	}

	responseDecode, ok := decodeResponse.(map[string]interface{})

	if !ok || len(responseDecode) == 0 {
		return nil, &SDError{"Response parsing error", result, res.StatusCode}
	}

	if val, ok := responseDecode["errors"]; ok {
		contentErrors := val.([]interface{})[0].(map[string]interface{})
		var moreInfo interface{}

		if (contentErrors["more-info"] != nil) {
			moreInfo = contentErrors["more-info"]
		} else {
			moreInfo = nil
		}

		return nil, &SDError{contentErrors["message"].(string), moreInfo, res.StatusCode}
	}
	if res.StatusCode >= 400 {
		return nil, &SDError{"Unknown Error", result, res.StatusCode}
	}
	return responseDecode, nil
}

// DecodeJSON converts string JSON to map[string]interface{}, its length is 0
// in case of JSON parsing error
func (s *SlicingDice) DecodeJSON(jsonData string) (interface{}, error) {
	var f interface{}
	d := json.NewDecoder(strings.NewReader(jsonData))
	err := d.Decode(&f)
	return f, err
}

// GetDatabase gets information about the current SlicingDice database
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetDatabase() (map[string]interface{}, error) {
	url := s.getFullUrl(DATABASE)
	return s.makeRequest(url, "GET", 2, nil)
}

// GetColumns get columns stored in your SlicingDice account
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetColumns() (map[string]interface{}, error) {
	url := s.getFullUrl(COLUMN)
	return s.makeRequest(url, "GET", 2, nil)
}

// GetSavedQuery get a saved query by name
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetSavedQuery(queryName string) (map[string]interface{}, error) {
	url := s.getFullUrl(SAVED + queryName)
	return s.makeRequest(url, "GET", 0, nil)
}

// DeleteSavedQuery delete a saved query by name
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) DeleteSavedQuery(queryName string) (map[string]interface{}, error) {
	url := s.getFullUrl(SAVED + queryName)
	return s.makeRequest(url, "DELETE", 2, nil)
}

// GetSavedQueries get all saved queryName
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetSavedQueries() (map[string]interface{}, error) {
	url := s.getFullUrl(SAVED)
	return s.makeRequest(url, "GET", 2, nil)
}

// Inserts data in a SlicingDice database.
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) Insert(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(INSERT)
	return s.makeRequest(url, "POST", 1, query)
}

// CreateColumn create a column in SlicingDice
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CreateColumn(query interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(COLUMN)
	validate := hasValidColumn(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 1, query)
}

// CountEntity makes a count entity query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEntity(query interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(COUNT_ENTITY)
	validate := hasValidCountQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// CountEntityTotal get total of entity query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEntityTotal(data ...[]string) (map[string]interface{}, error) {
	dimensions := make(map[string]interface{})

	if (len(data) != 0) {
		dimensions = map[string]interface{}{
			"dimensions": data[0],
		}
	}

	url := s.getFullUrl(COUNT_ENTITY_TOTAL)
	return s.makeRequest(url, "POST", 0, dimensions)
}

// CountEvent makes a count event query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEvent(query interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(COUNT_EVENT)
	validate := hasValidCountQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// Aggregation makes a aggregation query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) Aggregation(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(AGGREGATION)
	return s.makeRequest(url, "POST", 0, query)
}

// Result makes a data extraction result query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) Result(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(RESULT)
	validate := hasValidDataExtractionQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// Score makes a data extraction score query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) Score(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(SCORE)
	validate := hasValidDataExtractionQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// TopValues makes a top values query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) TopValues(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(TOP_VALUES)
	validate := hasValidTopValuesQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// ExistsEntity makes a exists entity query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) ExistsEntity(ids []string, dimension string) (map[string]interface{}, error) {
	url := s.getFullUrl(EXISTS_ENTITY)
	query := make(map[string]interface{})
	query["ids"] = ids
	if dimension != "" {
		query["dimension"] = dimension
	}
	return s.makeRequest(url, "POST", 0, query)
}

// CreateSavedQuery created a saved query in SlicingDice
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CreateSavedQuery(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(SAVED)
	validate := hasValidSavedQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 2, query)
}

// UpdateSavedQuery update a saved query in SlicingDice by name
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) UpdateSavedQuery(queryName string, query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(SAVED + queryName)
	return s.makeRequest(url, "PUT", 2, query)
}

func (s *SlicingDice) Sql(query string) (map[string]interface{}, error) {
	url := s.getFullUrl(SQL)
	return s.makeRequestSQL(url, "POST", 0, query, true)
}
