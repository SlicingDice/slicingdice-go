package main

import (
	"../slicingdice"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
	"reflect"
)

type SlicingDiceTester struct {
	client           *slicingdice.SlicingDice
	columnTranslation map[string]string
	sleepTime        int
	path             string
	extension        string
	numSuccess       int
	numFails         int
	failedTests      []string
	verbose          bool
}

// Run all the tests of determined query type
func (s *SlicingDiceTester) runTests(queryType string) {
	testData := s.loadTestData(queryType).([]interface{})
	numTests := len(testData)

	for i, test := range testData {
		var err error
		var result map[string]interface{}
		testConverted := test.(map[string]interface{})
		s.emptyColumnTranslation()

		fmt.Printf("(%v/%v) Executing test \"%v\"\n", i+1, numTests, testConverted["name"])

		if _, ok := testConverted["description"]; ok {
			fmt.Printf("  Description: %v\n", testConverted["description"])
		}

		fmt.Printf("  Query type: %v\n", queryType)
		err = s.createColumns(testConverted)
		if err != nil {
			s.compareResult(testConverted, nil, err)
			continue
		}
		err = s.insertData(testConverted)
		if err != nil {
			s.compareResult(testConverted, nil, err)
			continue
		}
		result, err = s.executeQuery(queryType, testConverted)
		if err != nil {
			s.compareResult(testConverted, nil, err)
			continue
		}

		s.compareResult(testConverted, result, nil)
	}
}

// Create columns on Slicing Dice API
func (s *SlicingDiceTester) createColumns(test map[string]interface{}) error {
	var columnOrColumns string
	columns := test["columns"].([]interface{})
	isSingular := len(columns) == 1

	if isSingular {
		columnOrColumns = "column"
	} else {
		columnOrColumns = "columns"
	}

	fmt.Printf("  Creating %v %v\n", len(columns), columnOrColumns)

	for _, column := range columns {
		newColumn := s.appendTimestampToColumnName(column.(map[string]interface{}))
		_, err := s.client.CreateColumn(newColumn)

		if err != nil {
			return err
		}

		if s.verbose {
			fmt.Printf("    - %v\n", newColumn["api-name"])
		}
	}
	return nil
}

/* Append a timestamp to column name
This technique allows the same test suite to be executed over and over
again, since each execution will use different column names.
*/
func (s *SlicingDiceTester) appendTimestampToColumnName(column map[string]interface{}) map[string]interface{} {
	oldName := fmt.Sprintf("\"%v\"", column["api-name"])

	timestamp := s.getTimestamp()
	column["name"] = column["name"].(string) + timestamp
	column["api-name"] = column["api-name"].(string) + timestamp
	newName := fmt.Sprintf("\"%v\"", column["api-name"])

	s.columnTranslation[oldName] = newName
	return column
}

// Get actual timestamp on string
func (s *SlicingDiceTester) getTimestamp() string {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%v", now)
}

// Erase column translation map
func (s *SlicingDiceTester) emptyColumnTranslation() {
	s.columnTranslation = map[string]string{}
}

// Insert data on Slicing Dice API
func (s *SlicingDiceTester) insertData(test map[string]interface{}) error {
	var entityOrEntities string
	insert := test["insert"].(map[string]interface{})
	isSingular := len(insert) == 1

	if isSingular {
		entityOrEntities = "entity"
	} else {
		entityOrEntities = "entities"
	}

	fmt.Printf("  Inserting %v %v\n", len(insert), entityOrEntities)

	insertDataTranslated := s.translateColumnNames(insert, true)

	if s.verbose {
		fmt.Printf("    - %v\n", insertDataTranslated)
	}

	_, err := s.client.Insert(insertDataTranslated)
	if err != nil {
		fmt.Println(err)
		return err
	}

	time.Sleep(time.Duration(s.sleepTime) * time.Second)

	return nil
}

// Translate column names to match the name with timestamp
func (s *SlicingDiceTester) translateColumnNames(jsonData map[string]interface{}, isRequest bool) map[string]interface{} {
	dataConverted, _ := json.Marshal(jsonData)
	dataString := string(dataConverted)

	for oldName, newName := range s.columnTranslation {
		dataString = strings.Replace(dataString, oldName, newName, -1)
	}

	if isRequest {
		return s.decodeWithNumberJSON(dataString).(map[string]interface{})
	} else {
		return s.client.DecodeJSON(dataString).(map[string]interface{})
	}
}

// Execute a query of a determined type on Slicing Dice API
func (s *SlicingDiceTester) executeQuery(queryType string, test map[string]interface{}) (map[string]interface{}, error) {
	var result interface{}
	var err error
	query := test["query"].(map[string]interface{})
	queryDataTranslated := s.translateColumnNames(query, true)

	fmt.Println("  Querying")
	if s.verbose {
		fmt.Printf("    - %v\n", queryDataTranslated)
	}

	if queryType == "count_entity" {
		result, err = s.client.CountEntity(queryDataTranslated)
	} else if queryType == "count_event" {
		result, err = s.client.CountEvent(queryDataTranslated)
	} else if queryType == "top_values" {
		result, err = s.client.TopValues(queryDataTranslated)
	} else if queryType == "aggregation" {
		result, err = s.client.Aggregation(queryDataTranslated)
	} else if queryType == "result" {
		result, err = s.client.Result(queryDataTranslated)
	} else if queryType == "score" {
		result, err = s.client.Score(queryDataTranslated)
	}
	if result == nil {
		return nil, err
	}
	return result.(map[string]interface{}), err
}

// Compare and assert result received from Slicing Dice API
func (s *SlicingDiceTester) compareResult(test map[string]interface{}, result map[string]interface{}, err error) {
	expected := s.translateColumnNames(test["expected"].(map[string]interface{}), false)
	if err != nil {
		s.numFails += 1
		s.failedTests = append(s.failedTests, test["name"].(string))
		expectedData, _ := json.Marshal(expected["result"])
		fmt.Printf("  Expected: \"%v\": %v\n", "result", string(expectedData))
		fmt.Printf("  Result: \"result\": %v\n", err)
		fmt.Println("  Status: Failed\n")
	} else if result != nil {
		for key, value := range expected {
			if value == "ignore" {
				continue
			}

			if !s.compareJSONValue(result[key], expected[key]) {
				resultData, _ := json.Marshal(result[key])
				expectedData, _ := json.Marshal(expected[key])

				s.numFails += 1
				s.failedTests = append(s.failedTests, test["name"].(string))

				fmt.Printf("  Expected: \"%v\": %v\n", key, string(expectedData))
				fmt.Printf("  Result: \"result\": %v\n", string(resultData))
				fmt.Println("  Status: Failed\n")
				return
			} else {
				s.numSuccess += 1
				fmt.Println("  Status: Passed\n")
			}
		}
	}
}

func (s *SlicingDiceTester) compareJSON(expected map[string]interface{}, got map[string]interface{}) bool {
	if len(expected) != len(got) {
		return false
	}

	for key, value := range expected {
		valueExpected := value
		valueGot := got[key]

		if !s.compareJSONValue(valueExpected, valueGot) {
			return false
		}
	}

	return true
}

func (s *SlicingDiceTester) compareJSONArray(expected []interface{}, got []interface{}) bool {
	if len(expected) != len(got) {
		return false
	}

	for i, value := range expected {
		valueExpected := value
		valueGot := got[i]

		if !s.compareJSONValue(valueExpected, valueGot) {
			return false
		}
	}

	return true
}

func (s *SlicingDiceTester) compareJSONValue(expected interface{}, got interface{}) bool {
	if reflect.ValueOf(expected).Kind() == reflect.Map {
		expectedMap := expected.(map[string]interface{})
		gotMap := got.(map[string]interface{})
		return s.compareJSON(expectedMap, gotMap)
	} else if reflect.ValueOf(expected).Kind() == reflect.Slice {
		expectedArray := expected.([]interface{})
		gotArray := got.([]interface{})
		return s.compareJSONArray(expectedArray, gotArray)
	} else {
		return reflect.DeepEqual(expected, got)
	}
}

// Load test data from examples folder
func (s *SlicingDiceTester) loadTestData(queryType string) interface{} {
	filename := s.path + queryType + s.extension
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	return s.decodeWithNumberJSON(string(file))
}

func (s *SlicingDiceTester) decodeWithNumberJSON(jsonData string) interface{} {
	var f interface{}
	d := json.NewDecoder(strings.NewReader(jsonData))
	d.UseNumber()
	if err := d.Decode(&f); err != nil {
		log.Fatal(err)
	}
	return f
}


func newSlicingDiceTester(apiKey string, verboseOption bool) (t *SlicingDiceTester) {
	keys := new(slicingdice.APIKey)
	keys.MasterKey = apiKey
	sdTester := new(SlicingDiceTester)
	sdTester.client = slicingdice.New(keys, 60)
	sdTester.client.Test = true
	sdTester.verbose = verboseOption

	// Sleep Time in seconds
	sdTester.sleepTime = 10
	// Path for examples
	sdTester.path = "examples/"
	// Examples files extension
	sdTester.extension = ".json"

	sdTester.numSuccess = 0
	sdTester.numFails = 0
	return sdTester
}

func printResults(sdTester *SlicingDiceTester) {
	fmt.Println("\nResults:")
	fmt.Println("  Successes:", sdTester.numSuccess)
	fmt.Println("  Fails:", sdTester.numFails)

	for _, failedTest := range sdTester.failedTests {
		fmt.Println("    - ", failedTest)
	}
	fmt.Println("")

	if sdTester.numFails > 0 {
		var testOrTests string
		isSingular := sdTester.numFails == 1

		if isSingular {
			testOrTests = "test has"
		} else {
			testOrTests = "tests have"
		}
		fmt.Printf("FAIL: %v %v failed\n", sdTester.numFails, testOrTests)
		os.Exit(1)
	} else {
		fmt.Println("SUCCESS: All tests passed")
		os.Exit(0)
	}
}

func main() {
	// SlicingDice queries to be tested. Must match the JSON file name.
	var queryTypes = [6]string{
		"count_entity",
		"count_event",
		"top_values",
		"aggregation",
		"result",
		"score",
	}

	// Testing class with demo API key
	// You can get a new demo API key here: http://panel.slicingdice.com/docs/#api-details-api-connection-api-keys-demo-key
	sdTester := newSlicingDiceTester(
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJfX3NhbHQiOiJkZW1vNzRtIiwicGVybWlzc2lvbl9sZXZlbCI6MywicHJvamVjdF9pZCI6MjM1LCJjbGllbnRfaWQiOjEwfQ.f9yLh6M82NX06r3TemFLmZ2U-tadBqgKF2EuONZrOK0",
		false,
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		printResults(sdTester)
	}()

	for _, queryType := range queryTypes {
		sdTester.runTests(queryType)
	}
	printResults(sdTester)
}
