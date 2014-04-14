// Package rest is a simple REST interface over MongoDB, as middlware for martini framework
//
// package main
//
// import (
//   "github.com/go-martini/martini"
//   "labix.org/v2/mgo"
//   "github.com/olebedev/rest"
// )
//
// func main() {
//   session, err := mgo.Dial("localhost")
//   if err != nil {
//     panic(err)
//   }
//   defer session.Close()
//   session.SetMode(mgo.Monotonic, true)
//   db := session.DB("test")
//
//   m := martini.Classic()
//
//   m.Group("/api/v1", rest.Rest(rest.Config{
//     Db            : db,
//     ResonseField  : "data", // optional
//     Autoincrement : true,   // optional
//   }))
//
//   m.Run()
// }
//
package rest

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/go-martini/martini"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
)

// Config is a struct for specifying configuration options for the rest.Rest middleware.
type Config struct {
	// Mongo instance pointer.
	Db *mgo.Database
	// Optional response field name. It is necessary to obtain the expected response.
	ResonseField string
	// Use integer autoincrement for _id instead of mongodb auto generated hash
	Autoincrement bool
}

var conf Config

// JSON maker. For flexible JSON data packing.
func jsonResponse(err error, data interface{}) string {
	var resp []byte
	if len(conf.ResonseField) > 0 {
		m := map[string]interface{}{}
		if err != nil {
			m["error"] = err.Error()
		} else {
			m[conf.ResonseField] = data
		}
		resp, _ = json.Marshal(m)
		return string(resp)
	}

	if err != nil {
		m := map[string]interface{}{
			"error": err.Error(),
		}
		resp, _ = json.Marshal(m)
	} else {
		resp, _ = json.Marshal(data)
	}

	return string(resp)
}

// Parse _id from url params. Intended to the mongodb query.
func parseId(s string) (_id interface{}, ok bool) {
	ok = true
	d, e := hex.DecodeString(s)
	if e != nil || len(d) != 12 {
		_id = s
		if len(s) == 0 {
			ok = false
			return
		}
		if v, err := strconv.Atoi(s); err == nil {
			_id = v
		}
	} else {
		_id = bson.ObjectIdHex(s)
	}
	return
}

func incrementFor(coll string) int {
	c := conf.Db.C("ids")
	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"n": 1}},
		ReturnNew: true,
	}

	var doc struct {
		N int
	}

	// check
	count, _ := c.Find(bson.M{"_id": coll}).Count()
	if count == 0 {
		c.Insert(bson.M{"_id": coll, "n": 0})
	}
	c.Find(bson.M{"_id": coll}).Apply(change, &doc)
	return doc.N
}

// GET method martini.Handler. For collections. With optional GET parameters.
func get(req *http.Request, params martini.Params) (int, string) {
	c := conf.Db.C(params["coll"])
	var query bson.M
	result := []interface{}{}
	if len(req.FormValue("query")) > 0 {
		err := json.Unmarshal([]byte(req.FormValue("query")), &query)
		if err != nil {
			query = nil
		}
	}

	q := c.Find(query)

	// Limit
	if len(req.FormValue("limit")) > 0 {
		v, err := strconv.Atoi(req.FormValue("limit"))
		if err == nil {
			q.Limit(v)
		}
	}

	// Skip
	if len(req.FormValue("skip")) > 0 {
		v, err := strconv.Atoi(req.FormValue("skip"))
		if err == nil {
			q.Skip(v)
		}
	}

	// Count
	if len(req.FormValue("count")) > 0 {
		count, err := q.Count()
		return http.StatusOK, jsonResponse(err, count)
	}

	// Sort
	if len(req.FormValue("sort")) > 0 {
		q.Sort(req.Form["sort"]...)
	}

	// Select
	var s bson.M
	if len(req.FormValue("select")) > 0 {
		err := json.Unmarshal([]byte(req.FormValue("select")), &s)
		if err == nil {
			q.Select(s)
		}
	}

	err := q.All(&result)
	return http.StatusOK, jsonResponse(err, result)
}

// GET method martini.Handler to get item by _id.
func getId(params martini.Params) (int, string) {
	c := conf.Db.C(params["coll"])
	_id, _ := parseId(params["_id"])
	q := c.Find(bson.M{"_id": _id})
	var result interface{}
	err := q.One(&result)
	if err != nil {
		if err.Error() == "not found" {
			return http.StatusNotFound, jsonResponse(err, result)
		} else {
			return http.StatusBadRequest, jsonResponse(err, result)
		}
	}
	return http.StatusOK, jsonResponse(err, result)
}

// POST method martini.Handler. To create item.
func post(req *http.Request, params martini.Params) (int, string) {
	c := conf.Db.C(params["coll"])

	// parse body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}
	var body bson.M
	err = json.Unmarshal(b, &body)
	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}

	// m, _ := body.(map[string]interface{})

	if _, exists := body["_id"]; !exists {
		if conf.Autoincrement {
			body["_id"] = incrementFor(params["coll"])
		} else {
			body["_id"] = bson.NewObjectId()
		}
	}

	err = c.Insert(body)

	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}
	return http.StatusCreated, jsonResponse(err, body)
}

// PUT method martini.Handler. To replace item by _id.
func put(req *http.Request, params martini.Params) (int, string) {
	c := conf.Db.C(params["coll"])
	response := map[string]int{"updated": 1}
	_id, ok := parseId(params["_id"])
	// not ok if _id == ""
	if !ok {
		return http.StatusBadRequest, jsonResponse(errors.New("invalid _id"), nil)
	}

	count, err := c.Find(bson.M{"_id": _id}).Count()
	response["updated"] = count
	if count == 0 {
		return http.StatusNotFound, jsonResponse(errors.New("not found"), nil)
	}

	// parse body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}
	var body interface{}
	err = json.Unmarshal(b, &body)
	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}

	v, _ := body.(map[string]interface{})
	delete(v, "_id")

	err = c.UpdateId(_id, v)
	if err != nil {
		return http.StatusBadRequest, jsonResponse(err, nil)
	}

	return http.StatusOK, jsonResponse(err, response)
}

// DELETE method martini.Handler. To delete item by _id.
func del(req *http.Request, params martini.Params) (int, string) {
	c := conf.Db.C(params["coll"])
	_id, ok := parseId(params["_id"])
	response := map[string]int{"removed": 1}
	if !ok {
		return http.StatusBadRequest, jsonResponse(errors.New("invalid _id"), nil)
	}

	err := c.RemoveId(_id)
	if err != nil {
		if err.Error() == "not found" {
			return http.StatusNotFound, jsonResponse(err, nil)
		} else {
			return http.StatusBadRequest, jsonResponse(err, nil)
		}
	}
	return http.StatusOK, jsonResponse(err, response)
}

func Rest(c Config) func(r martini.Router) {
	conf = c
	return func(r martini.Router) {
		r.Get("/:coll", get)
		r.Get("/:coll/:_id", getId)
		r.Post("/:coll", post)
		r.Put("/:coll/:_id", put)
		r.Delete("/:coll/:_id", del)
	}
}
