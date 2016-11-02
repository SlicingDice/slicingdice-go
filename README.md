# SlicingDice Official Go Client (v1.0)

Official Go client for [SlicingDice](http://www.slicingdice.com/), Data Warehouse and Analytics Database as a Service.

## Documentation

If you are new to SlicingDice, check our [quickstart guide](http://panel.slicingdice.com/docs/#quickstart-guide) and learn to use it in 15 minutes.

Please refer to the [SlicingDice official documentation](http://panel.slicingdice.com/docs/) for more information on [analytics databases](http://panel.slicingdice.com/docs/#analytics-concepts), [data modeling](http://panel.slicingdice.com/docs/#data-modeling), [indexing](http://panel.slicingdice.com/docs/#data-indexing), [querying](http://panel.slicingdice.com/docs/#data-querying), [limitations](http://panel.slicingdice.com/docs/#current-slicingdice-limitations) and [API details](http://panel.slicingdice.com/docs/#api-details).

## Tests and Examples

Whether you want to test the client installation or simply check more examples on how the client works, take a look at [tests and examples directory](tests_and_examples/).

## Installing

In order to install the Go client, you only need to execute the [`go get`](https://golang.org/cmd/go/#hdr-Download_and_install_packages_and_dependencies) command.

```bash
go get github.com/SlicingDice/slicingdice-go
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice-go"
)

func main() {
    // Configure client
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "mykey"
    client := slicingdice.New(keys, 60)

    // Indexing data
    index_data := map[string]interface{}{
        "user1@slicingdice.com": map[string]int{
            "age": 22,
        },
        "auto-create-fields": true,
    }
    client.Index(index_data)

    // Querying data
    query_data := map[string]interface{}{
        "users-between-20-and-40": map[string]interface{}{
            "age": map[string][]int{
                "range": []int{20, 40},
            },
        },
    }
    result, _ := client.CountEntity(query_data)
    fmt.Println(result["status"])
}
```

## Reference

`SlicingDice` encapsulates logic for sending requests to the API. Its methods are thin layers around the [API endpoints](http://panel.slicingdice.com/docs/#api-details-api-endpoints), so their parameters and return values are JSON-like `interface{}` objects with the same syntax as the [API endpoints](http://panel.slicingdice.com/docs/#api-details-api-endpoints)

### Attributes

* `key (map[string]string)` - [API key](http://panel.slicingdice.com/docs/#api-details-api-connection-api-keys) to authenticate requests with the SlicingDice API.
* `timeout (int)` - Amount of time, in seconds, to wait for results for each request.

### Constructors

`New(key *APIKey, timeout int) *SlicingDice`
* `key (APIKey)` - [API key](http://panel.slicingdice.com/docs/#api-details-api-connection-api-keys) to authenticate requests with the SlicingDice API.
* `timeout (int)` - Amount of time, in seconds, to wait for results for each request.

### `GetProjects()`
Get all created projects, both active and inactive ones. This method corresponds to a [GET request at /project](http://panel.slicingdice.com/docs/#api-details-api-endpoints-get-project).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.GetProjects())
}
```

#### Output example

```json
{
    "active": [
        {
            "name": "Project 1",
            "description": "My first project",
            "data-expiration": 30,
            "created-at": "2016-04-05T10:20:30Z"
        }
    ],
    "inactive": [
        {
            "name": "Project 2",
            "description": "My second project",
            "data-expiration": 90,
            "created-at": "2016-04-05T10:20:30Z"
        }
    ]
}
```

### `GetFields()`
Get all created fields, both active and inactive ones. This method corresponds to a [GET request at /field](http://panel.slicingdice.com/docs/#api-details-api-endpoints-get-field).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.GetFields())
}
```

#### Output example

```json
{
    "active": [
        {
          "name": "Model",
          "api-name": "car-model",
          "description": "Car models from dealerships",
          "type": "string",
          "category": "general",
          "cardinality": "high",
          "storage": "latest-value"
        }
    ],
    "inactive": [
        {
          "name": "Year",
          "api-name": "car-year",
          "description": "Year of manufacture",
          "type": "integer",
          "category": "general",
          "storage": "latest-value"
        }
    ]
}
```

### `CreateField(query interface{})`
Create a new field. This method corresponds to a [POST request at /field](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-field).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    fieldData := map[string]interface{}{
        "name":        "Year",
        "type":        "integer",
        "description": "Year of manufacturing",
        "storage":     "lastest-value",
    }
    fmt.Println(sd.CreateField(fieldData))
}
```

#### Output example

```json
{
    "status": "success",
    "api-name": "year"
}
```

### `Index(query interface{})`
Index data to existing entities or create new entities, if necessary. This method corresponds to a [POST request at /index](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-index).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    indexData := map[string]interface{}{
        "user1@slicingdice.com": map[string]interface{}{
            "car-model": "Ford Ka",
            "year":      2016,
        },
        "user2@slicingdice.com": map[string]interface{}{
            "car-model": "Honda Fit",
            "year":      2016,
        },
        "user3@slicingdice.com": map[string]interface{}{
            "car-model": "Toyota Corolla",
            "year":      2010,
            "test-drives": []map[string]string{
                map[string]string{
                    "value": "NY",
                    "date":  "2016-08-17T13:23:47+00:00",
                },
                map[string]string{
                    "value": "NY",
                    "date":  "2016-08-17T13:23:47+00:00",
                },
                map[string]string{
                    "value": "CA",
                    "date":  "2016-04-05T10:20:30Z",
                },
            },
        },
        "user4@slicingdice.com": map[string]interface{}{
            "car-model": "Ford Ka",
            "year":      2005,
            "test-drives": []map[string]string{
                map[string]string{
                    "value": "NY",
                    "date":  "2016-08-17T13:23:47+00:00",
                },
            },
        },
    }
    fmt.Println(sd.Index(indexData))
}
```

#### Output example

```json
{
    "status": "success",
    "indexed-entities": 4,
    "indexed-fields": 10,
    "took": 0.023
}
```

### `ExistsEntity(ids)`
Verify which entities exist in a project given a list of entity IDs. This method corresponds to a [POST request at /query/exists/entity](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-exists-entity).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    entities := []string{
        "user1@slicingdice.com",
        "user2@slicingdice.com",
        "user3@slicingdice.com",
    }
    fmt.Println(sd.ExistsEntity(entities))
}
```

#### Output example

```json
{
    "status": "success",
    "exists": [
        "user1@slicingdice.com",
        "user2@slicingdice.com"
    ],
    "not-exists": [
        "user3@slicingdice.com"
    ],
    "took": 0.103
}
```

### `CountEntityTotal()`
Count the number of indexed entities. This method corresponds to a [GET request at /query/count/entity/total](http://panel.slicingdice.com/docs/#api-details-api-endpoints-get-query-count-entity-total).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.CountEntityTotal())
}
```

#### Output example

```json
{
    "status": "success",
    "result": {
        "total": 42
    },
    "took": 0.103
}
```

### `CountEntity(query interface{})`
Count the number of entities attending the given query. This method corresponds to a [POST request at /query/count/entity](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-count-entity).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "users-in-ny-or-ca": []interface{}{
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "NY",
                },
            },
            "or",
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "CA",
                },
            },
        },
        "users-in-fl": map[string]interface{}{
            "state": map[string]string{
                "equals": "NY",
            },
        },
        "bypass-cache": false,
    }
    fmt.Println(sd.CountEntity(query))
}
```

#### Output example

```json
{
    "status": "success",
    "result": {
        "users-from-ny-or-ca": 175,
        "users-from-ny": 296
    },
    "took": 0.103
}
```

### `CountEvent(query interface{})`
Count the number of occurrences for time-series events attending the given query. This method corresponds to a [POST request at /query/count/event](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-count-event).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "users-from-ny-in-jan": []map[string]interface{}{
            map[string]interface{}{
                "test-field": []map[string]interface{}{
                    map[string]interface{}{
                        "equals": "NY",
                        "between": []string{
                            "2016-04-01T00:00:00Z",
                            "2016-04-03T00:00:00Z",
                        },
                        "minfreq": 2,
                    },
                },
            },
        },
        "users-from-ny-in-feb": []map[string]interface{}{
            map[string]interface{}{
                "test-field": []map[string]interface{}{
                    map[string]interface{}{
                        "equals": "NY",
                        "between": []string{
                            "2016-02-01T00:00:00Z",
                            "2016-02-28T00:00:00Z",
                        },
                        "minfreq": 2,
                    },
                },
            },
        },
        "bypass-cache": true,
    }
    fmt.Println(sd.CountEvent(query))
}
```

#### Output example

```json
{
    "status": "success",
    "result": {
        "users-from-ny-in-jan": 175,
        "users-from-ny-in-feb": 296
    },
    "took": 0.103
}
```

### `TopValues(query interface{})`
Return the top values for entities attending the given query. This method corresponds to a [POST request at /query/top_values](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-top-values).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "user-gender": map[string]interface{}{
            "gender": 2,
        },
        "operating-systems": map[string]interface{}{
            "os": 3,
        },
        "linux-operating-systems": map[string]interface{}{
            "os": 3,
            "contains": []string{
                "linux",
                "unix",
            },
        },
    }
    fmt.Println(sd.TopValues(query))
}
```

#### Output example

```json
{
    "status": "success",
    "result": {
        "user-gender": {
            "gender": [
                {
                    "quantity": 6.0,
                    "value": "male"
                }, {
                    "quantity": 4.0,
                    "value": "female"
                }
            ]
        },
        "operating-systems": {
            "os": [
                {
                    "quantity": 55.0,
                    "value": "windows"
                }, {
                    "quantity": 25.0,
                    "value": "macos"
                }, {
                    "quantity": 12.0,
                    "value": "linux"
                }
            ]
        },
        "linux-operating-systems": {
            "os": [
                {
                    "quantity": 12.0,
                    "value": "linux"
                }, {
                    "quantity": 3.0,
                    "value": "debian-linux"
                }, {
                    "quantity": 2.0,
                    "value": "unix"
                }
            ]
        }
    },
    "took": 0.103
}
```

### `Aggregation(query interface{})`
Return the aggregation of all fields in the given query. This method corresponds to a [POST request at /query/aggregation](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-aggregation).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "query": []map[string]interface{}{
            map[string]interface{}{
                "gender": 2,
            },
            map[string]interface{}{
                "os": 2,
                "equals": []string{
                    "linux",
                    "macos",
                    "windows",
                },
            },
            map[string]interface{}{
                "browser": 2,
            },
        },
    }
    fmt.Println(sd.Aggregation(query))
}
```

#### Output example

```json
{
    "status": "success",
    "result": {
        "gender": [
            {
                "quantity": 6,
                "value": "male",
                "os": [
                    {
                        "quantity": 5,
                        "value": "windows",
                        "browser": [
                            {
                                "quantity": 3,
                                "value": "safari"
                            }, {
                                "quantity": 2,
                                "value": "internet explorer"
                            }
                        ]
                    }, {
                        "quantity": 1,
                        "value": "linux",
                        "browser": [
                            {
                                "quantity": 1,
                                "value": "chrome"
                            }
                        ]
                    }
                ]
            }, {
                "quantity": 4,
                "value": "female",
                "os": [
                    {
                        "quantity": 3,
                        "value": "macos",
                        "browser": [
                            {
                                "quantity": 3,
                                "value": "chrome"
                            }
                        ]
                    }, {
                        "quantity": 1,
                        "value": "linux",
                        "browser": [
                            {
                                "quantity": 1,
                                "value": "chrome"
                            }
                        ]
                    }
                ]
            }
        ]
    },
    "took": 0.103
}
```

### `GetSavedQueries()`
Get all saved queries. This method corresponds to a [GET request at /query/saved](http://panel.slicingdice.com/docs/#api-details-api-endpoints-get-query-saved).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.GetSavedQueries())
}
```

#### Output example

```json
{
    "status": "success",
    "saved-queries": [
        {
            "name": "users-in-ny-or-from-ca",
            "type": "count/entity",
            "query": [
                {
                    "state": {
                        "equals": "NY"
                    }
                },
                "or",
                {
                    "state-origin": {
                        "equals": "CA"
                    }
                }
            ],
            "cache-period": 100
        }, {
            "name": "users-from-ca",
            "type": "count/entity",
            "query": [
                {
                    "state": {
                        "equals": "NY"
                    }
                }
            ],
            "cache-period": 60
        }
    ],
    "took": 0.103
}
```

### `CreateSavedQuery(query interface{})`
Create a saved query at SlicingDice. This method corresponds to a [POST request at /query/saved](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-query-saved).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "name": "my-saved-query",
        "type": "count/entity",
        "query": []interface{}{
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "NY",
                },
            },
            "or",
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "CA",
                },
            },
        },
        "cache-period": 100,
    }
    fmt.Println(sd.CreateSavedQuery(query))
}
```

#### Output example

```json
{
    "status": "success",
    "name": "my-saved-query",
    "type": "count/entity",
    "query": [
        {
            "state": {
                "equals": "NY"
            }
        },
        "or",
        {
            "state-origin": {
                "equals": "CA"
            }
        }
    ],
    "cache-period": 100,
    "took": 0.103
}
```

### `UpdateSavedQuery(string queryName, query interface{})`
Update an existing saved query at SlicingDice. This method corresponds to a [PUT request at /query/saved/QUERY_NAME](http://panel.slicingdice.com/docs/#api-details-api-endpoints-put-query-saved-query-name).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "type": "count/entity",
        "query": []interface{}{
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "NY",
                },
            },
            "or",
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "CA",
                },
            },
        },
        "cache-period": 100,
    }
    fmt.Println(sd.UpdateSavedQuery("my-saved-query", query))
}
```

#### Output example

```json
{
    "status": "success",
    "name": "my-saved-query",
    "type": "count/entity",
    "query": [
        {
            "state": {
                "equals": "NY"
            }
        },
        "or",
        {
            "state-origin": {
                "equals": "CA"
            }
        }
    ],
    "cache-period": 100,
    "took": 0.103
}
```

### `GetSavedQuery(string queryName)`
Executed a saved query at SlicingDice. This method corresponds to a [GET request at /query/saved/QUERY_NAME](http://panel.slicingdice.com/docs/#api-details-api-endpoints-get-query-saved-query-name).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.GetSavedQuery("my-saved-query"))
}
```

#### Output example

```json
{
    "status": "success",
    "type": "count/entity",
    "query": [
        {
            "state": {
                "equals": "NY"
            }
        },
        "or",
        {
            "state-origin": {
                "equals": "CA"
            }
        }
    ],
    "result": {
        "my-saved-query": 175
    },
    "took": 0.103
}
```

### `DeleteSavedQuery(string queryName)`
Delete a saved query at SlicingDice. This method corresponds to a [DELETE request at /query/saved/QUERY_NAME](http://panel.slicingdice.com/docs/#api-details-api-endpoints-delete-query-saved-query-name).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_API_KEY"
    sd := slicingdice.New(keys, 60)
    fmt.Println(sd.DeleteSavedQuery("my-saved-query"))
}
```

#### Output example

```json
{
    "status": "success",
    "deleted-query": "my-saved-query",
    "type": "count/entity",
    "query": [
        {
            "state": {
                "equals": "NY"
            }
        },
        "or",
        {
            "state-origin": {
                "equals": "CA"
            }
        }
    ],
    "took": 0.103
}
```

### `Result(query interface{})`
Retrieve indexed values for entities attending the given query. This method corresponds to a [POST request at /data_extraction/result](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-data-extraction-result).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "query": []interface{}{
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "NY",
                },
            },
            "or",
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "CA",
                },
            },
        },
        "fields":     []string{"name", "year"},
        "limit":      2,
    }
    fmt.Println(sd.Result(query))
}
```

#### Output example

```json
{
    "status": "success",
    "data": {
        "user1@slicingdice.com": {
            "name": "John",
            "year": 2016
        },
        "user2@slicingdice.com": {
            "name": "Mary",
            "year": 2005
        }
    },
    "took": 0.103
}
```

### `Score(query interface{})`
Retrieve indexed values as well as their relevance for entities attending the given query. This method corresponds to a [POST request at /data_extraction/score](http://panel.slicingdice.com/docs/#api-details-api-endpoints-post-data-extraction-score).

#### Request example

```go
package main

import (
    "fmt"
    "github.com/SlicingDice/slicingdice"
)

func main() {
    keys := new(slicingdice.APIKey)
    keys.MasterKey = "MASTER_OR_READ_API_KEY"
    sd := slicingdice.New(keys, 60)
    query := map[string]interface{}{
        "query": []interface{}{
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "NY",
                },
            },
            "or",
            map[string]interface{}{
                "state": map[string]string{
                    "equals": "CA",
                },
            },
        },
        "fields":     []string{"name", "year"},
        "limit":      2,
    }
    fmt.Println(sd.Score(query))
}
```

#### Output example

```json
{
    "status": "success",
    "data": {
        "user1@slicingdice.com": {
            "name": "John",
            "year": 2016,
            "score": 2
        },
        "user2@slicingdice.com": {
            "name": "Mary",
            "year": 2005,
            "score": 1
        }
    },
    "took": 0.103
}
```

## License

[MIT](https://opensource.org/licenses/MIT)
