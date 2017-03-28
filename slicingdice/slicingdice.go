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
)

var sd_base = os.Getenv("SD_API_ADDRESS")

// All these constants representing all endpoints available in SlicingDice API.
const (
	RESULT             = "/data_extraction/result/"
	SCORE              = "/data_extraction/score/"
	INDEX              = "/index/"
	FIELD              = "/field/"
	PROJECT            = "/project/"
	TOP_VALUES         = "/query/top_values/"
	EXISTS_ENTITY      = "/query/exists/entity/"
	COUNT_ENTITY       = "/query/count/entity/"
	COUNT_ENTITY_TOTAL = "/query/count/entity/total/"
	COUNT_EVENT        = "/query/count/event/"
	AGGREGATION        = "/query/aggregation/"
	SAVED              = "/query/saved/"
)

/* APIKey is used to access the keys that we insert in the SlicingDice API.
There is only one rule: if you put the master key, you do not put the other,
because already by default the client uses for everything. Otherwise,
use a key to writing, if you want to write (index data, create fields)
and use a key to reading (make queries).
*/
type APIKey struct {
	WriteKey  string
	ReadKey   string
	MasterKey string
	CustomKey string
}

// SlicingDice is the main structure of slicingdice. Through it, we will make queries,
// we will create fields, we'll take projects, etc.
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
	queryConverted := query.(map[string]interface{})
	if len(queryConverted) > 10 {
		return errors.New("Count Query Validator: the query count entity has a limit of 10 queries by request.")
	}
	return nil
}

// hasValidTopValuesQuery checks whether the top values query passed by user is valid. It
// validates especially if the query len is less than 5 and field len is less than 6.
func hasValidTopValuesQuery(query interface{}) error {
	queryConverted := query.(map[string]interface{})
	// check query limit
	if len(queryConverted) > 5 {
		return errors.New("Top Values Validator: the top values query has a limit of 5 queries by request.")
	}
	// check field limit
	for _, value := range queryConverted {
		if len(value.(map[string]interface{})) > 6 {
			return errors.New("Top Values Validator: the query exceeds the limit of fields per query in request")
		}
	}
	return nil
}

// hasValidDataExtractionQuery checks whether the data extraction(result and score)
// query passed by user is valid. It validates especially if the 'limit' key
// has a len less than 100 and if has a valid field.
func hasValidDataExtractionQuery(query interface{}) error {
	queryConverted := query.(map[string]interface{})
	if val, ok := queryConverted["limit"]; ok {
		if val.(int) > 100 {
			return errors.New("Data Extraction Validator: the field 'limit' has a value max of 100.")
		}
	}
	if val, ok := queryConverted["fields"]; ok {
		fields := reflect.ValueOf(val)
		if fields.Len() > 10 {
			return errors.New("Data Extraction Validator: The key 'fields' in data extraction result must have up to 10 fields.")
		}
	}
	return nil
}

// hasValidField checks whether the new field is valid. Checks type, name,
// description; enumerate, decimal-place and string types.
func hasValidField(query interface{}) error {
	if reflect.ValueOf(query).Kind() == reflect.Slice {
		fieldData := query.([]interface{})
		for _, field := range fieldData {
			field := field.(map[string]interface{})
	        validateField(field)
	    }
	} else {
		query := query.(map[string]interface{})
		validateField(query)
	}
	return nil
}

func validateField(query map[string]interface{}) error {
	validTypeFields := []string{
		"unique-id", "boolean", "string", "integer", "decimal",
		"enumerated", "date", "integer-time-series",
		"decimal-time-series", "string-time-series",
	}
	// validate name
	if _, ok := query["name"]; !ok {
		return errors.New("Field Validator: the field should have a name.")
	}
	name := query["name"]
	if len(name.(string)) > 80 {
		return errors.New("Field Validator: the field's name have a very big name.(Max: 80 chars)")
	}
	// validate description
	if _, ok := query["description"]; ok {
		description := query["description"]
		if len(description.(string)) > 300 {
			return errors.New("Field Validator: the field's description have a very big name.(Max: 300chars)")
		}
	}
	// validate type field
	if _, ok := query["type"]; !ok {
		return errors.New("Field Validator: the field should have a type.")
	}
	typeField := query["type"].(string)
	if !stringInSlice(typeField, validTypeFields) {
		return errors.New("Field Validator: this field have a invalid type.")
	}
	// validate decimal place key
	if _, ok := query["decimal-place"]; ok {
		decimalTypes := []string{"decimal", "decimal-time-series"}
		if !stringInSlice(query["type"].(string), decimalTypes) {
			return errors.New("Field Validator: this field is only accepted on type 'decimal' or 'decimal-time-series'.")
		}
	}
	// validate string field type
	if query["type"] == "string" {
		if _, ok := query["cardinality"]; !ok {
			return errors.New("Field Validator: the field with type string should have 'cardinality' key.")
		}
		cardinalityTypes := []string{"high", "low"}
		if !stringInSlice(query["cardinality"].(string), cardinalityTypes) {
			return errors.New("Field Validator: the field 'cardinality' has invalid value.")
		}
	}
	// validate enumerated field
	if query["type"] == "enumerated" {
		if _, ok := query["range"]; !ok {
			return errors.New("Field Validator: the 'enumerate' type needs of the 'range' parameter.")
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
	SlicingDice.Test = false
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
		if s.Test {
			return sd_base + "/test" + path
		}
		return sd_base + path
	} else {
		if s.Test {
			return "https://api.slicingdice.com/v1/test" + path
		}
	}
	return "https://api.slicingdice.com/v1" + path
}

// makeRequest checks request method, convert the query passed for use to JSON
// and executes the request.
func (s *SlicingDice) makeRequest(url string, method string, endpointKeyLevel int, query interface{}) (map[string]interface{}, error) {
	queryData := new(bytes.Buffer)
	json.NewEncoder(queryData).Encode(query)
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
	request, err := http.NewRequest(method, url, queryData)
	request.Header.Add("Authorization", key)
	request.Header.Add("Content-Type", "application/json")
	res, err := client.Do(request)
	return s.handlerResponse(res, err)
}

type SDError struct {
	message string
	moreInfo string
	code int
}

func (e *SDError) Error() string { 
	return fmt.Sprintf("Error Code: %d \nMessage: %s \nMore Info: %s", e.code, e.message, e.moreInfo)
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
	responseDecode := s.decodeJSON(result)
	if len(responseDecode) == 0 {
		return nil, &SDError{"Internal Error", "Nothing", res.StatusCode}
	}
	if val, ok := responseDecode["errors"]; ok {
		contentErrors := val.([]interface{})[0].(map[string]interface{})
		moreInfo := "Nothing"

		if (contentErrors["more-info"] != nil) {
			moreInfo = contentErrors["more-info"].(string)
		}

		return nil, &SDError{contentErrors["message"].(string), moreInfo, res.StatusCode}
	}
	if res.StatusCode >= 400 {
		return nil, &SDError{"Unknown Error", "Nothing", res.StatusCode}
	}
	return responseDecode, nil
}

// decodeJSON converts string JSON to map[string]interface{}, its length is 0
// in case of JSON parsing error
func (s *SlicingDice) decodeJSON(jsonData string) map[string]interface{} {
	var f interface{}
	var m map[string]interface{}
	b := []byte(jsonData)
	json.Unmarshal(b, &f)

	if f != nil {
		m = f.(map[string]interface{})
	}

	return m
}

// Project get all projects in your SlicingDice account
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetProjects() (map[string]interface{}, error) {
	url := s.getFullUrl(PROJECT)
	return s.makeRequest(url, "GET", 2, nil)
}

// GetFields get fields stored in your SlicingDice account
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) GetFields() (map[string]interface{}, error) {
	url := s.getFullUrl(FIELD)
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

// Index a collection of data in SlicingDice.
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) Index(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(INDEX)
	return s.makeRequest(url, "POST", 1, query)
}

// CreateField create a field in SlicingDice
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CreateField(query interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(FIELD)
	validate := hasValidField(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 1, query)
}

// CountEntity makes a count entity query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEntity(query map[string]interface{}) (map[string]interface{}, error) {
	url := s.getFullUrl(COUNT_ENTITY)
	validate := hasValidCountQuery(query)
	if validate != nil {
		return nil, validate
	}
	return s.makeRequest(url, "POST", 0, query)
}

// CountEntityTotal get total of entity query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEntityTotal() (map[string]interface{}, error) {
	url := s.getFullUrl(COUNT_ENTITY_TOTAL)
	return s.makeRequest(url, "GET", 0, nil)
}

// CountEvent makes a count event query
// It returns a JSON converted in map[string]interface{}
func (s *SlicingDice) CountEvent(query map[string]interface{}) (map[string]interface{}, error) {
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
func (s *SlicingDice) ExistsEntity(ids []string) (map[string]interface{}, error) {
	url := s.getFullUrl(EXISTS_ENTITY)
	query := make(map[string]interface{})
	query["ids"] = ids
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
