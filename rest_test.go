package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"testing"
)

// 1
func TestConfig(t *testing.T) {
	defer dropDb()
	m := martini.Classic()

	// check that db is ok
	db = getDb()
	if db == nil {
		// fail
		t.Error("Falied to initialize MongoDB.")
	}

	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_config", nil)

	m.ServeHTTP(res, req)
	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":[]}`)
}

// 2
func TestConfigResonseField(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "test", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_config", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"test":[]}`)
}

// 3
func TestInvalidPost(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test_create", bytes.NewBufferString("string"))

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusBadRequest)
	expect(t, res.Body.String(), `{"error":"invalid character 's' looking for beginning of value"}`)
}

// 4
func TestPost(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test_create", bytes.NewBufferString(`{"foo":"bar"}`))

	m.ServeHTTP(res, req)

	resp := res.Body.String()
	expect(t, res.Code, http.StatusCreated)

	var body map[string]string
	json.Unmarshal([]byte(resp), &body)

	res = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/test_create/"+body["_id"], nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)

	resp = res.Body.String()
	expect(t, res.Code, http.StatusOK)

	var body2 map[string]interface{}
	json.Unmarshal([]byte(resp), &body2)

	item, _ := body2["data"].(map[string]string)
	expect(t, body["_id"], item["_id"])
}

// 5
func TestPostIntId(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test_create", bytes.NewBufferString(`{"_id":1,"foo":"bar"}`))

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusCreated)
	expect(t, res.Body.String(), `{"data":{"_id":1,"foo":"bar"}}`)
}

// 6
func TestCollection(t *testing.T) {
	defer dropDb()
	generateData(3, "test_collection")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_collection", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":[{"_id":1,"foo":"bar-3"},{"_id":2,"foo":"bar-2"},{"_id":3,"foo":"bar-1"}]}`)
}

// 7
func TestQuery(t *testing.T) {
	defer dropDb()
	generateData(3, "test_query")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", `/api/v1/test_query?query={"foo":"bar-2"}`, nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":[{"_id":2,"foo":"bar-2"}]}`)
}

// 8
func TestLimitSkipSortSelect(t *testing.T) {
	defer dropDb()
	generateData(20, "test_limit_skip_sort_select")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", `/api/v1/test_limit_skip_sort_select?sort=_id&limit=3&skip=3&select={"_id":0}`, nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":[{"foo":"bar-17"},{"foo":"bar-16"},{"foo":"bar-15"}]}`)
}

// 9
func TestCount(t *testing.T) {
	defer dropDb()
	generateData(3, "test_count")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_count?count=1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":3}`)
}

// 10
func TestNotFound(t *testing.T) {
	defer dropDb()
	generateData(3, "test_not_found")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_not_found/4", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusNotFound)
	expect(t, res.Body.String(), `{"error":"not found"}`)
}

// 11
func TestInvalidPut(t *testing.T) {
	defer dropDb()
	generateData(1, "test_i_put")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_i_put/1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":{"_id":1,"foo":"bar-1"}}`)

	res = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/v1/test_i_put/1", bytes.NewBufferString(`{"_id":1,"foo":"bar-2","test":1}}`))

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusBadRequest)
	expect(t, res.Body.String(), `{"error":"invalid character '}' after top-level value"}`)

}

// 12
func TestPut(t *testing.T) {
	defer dropDb()
	generateData(1, "test_put")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_put/1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":{"_id":1,"foo":"bar-1"}}`)

	res = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/v1/test_put/1", bytes.NewBufferString(`{"_id":1,"foo":"bar-2","test":1}`))

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":{"updated":1}}`)

	res = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/test_put/1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":{"_id":1,"foo":"bar-2","test":1}}`)

}

// 13
func TestDelete(t *testing.T) {
	defer dropDb()
	generateData(1, "test_delete")
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/test_delete/1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `{"data":{"removed":1}}`)

	res = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/test_delete/1", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusNotFound)
	expect(t, res.Body.String(), `{"error":"not found"}`)

}

// 14
func TestApplyHandler(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}), func() (int, string) {
		return http.StatusForbidden, "test forbidden"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test_apply", nil)

	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusForbidden)
	expect(t, res.Body.String(), `test forbidden`)
}

// 15
func TestAutoIncrement(t *testing.T) {
	defer dropDb()
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", true}))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test_autoinc", bytes.NewBufferString(`{"foo":"bar"}`))
	m.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusCreated)
	expect(t, res.Body.String(), `{"data":{"_id":1,"foo":"bar"}}`)
}

// helpers
var db *mgo.Database
var wg sync.WaitGroup
var once sync.Once

func generateData(n int, name string) {
	m := martini.Classic()
	m.Group("/api/v1", Rest(Config{getDb(), "data", false}))

	for i := range make([]int, n) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/"+name,
			bytes.NewBufferString(
				fmt.Sprintf(`{"_id":%d,"foo":"bar-%d"}`, i+1, n-i)))
		m.ServeHTTP(res, req)
	}
}

func dropDb() {
	wg.Done()
	go func() {
		once.Do(func() {
			wg.Wait()
			// db.DropDatabase()
		})
	}()
}

func getDb() *mgo.Database {
	if db == nil {
		wg.Add(15) // test cases counter
		var mongoUri string
		if len(os.Getenv("WERCKER_MONGODB_HOST")) > 0 {
			mongoUri = os.Getenv("WERCKER_MONGODB_HOST") + ":" + os.Getenv("WERCKER_MONGODB_PORT")
		} else {
			mongoUri = "localhost"
		}
		session, err := mgo.Dial(mongoUri)
		if err != nil {
			fmt.Println(err)
		}

		db = session.DB("_rest_test")
		db.DropDatabase()
	}
	return db
}

func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

// API prefix for routing it right.
