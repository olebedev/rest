package rest

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/codegangsta/martini"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
)

type Config struct {
	Prefix       string
	Db           *mgo.Database
	ResonseField string
}

var conf *Config

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
		return 200, jsonResponse(err, count)
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
	return 200, jsonResponse(err, result)
}

func getId(params martini.Params) (int, string) {
	status := 200
	c := conf.Db.C(params["coll"])
	_id, _ := parseId(params["_id"])
	q := c.Find(bson.M{"_id": _id})
	var result interface{}
	err := q.One(&result)
	if err != nil {
		status = 400
		if err.Error() == "not found" {
			status = 404
		}
	}
	return status, jsonResponse(err, result)
}

func post(req *http.Request, params martini.Params) (int, string) {
	status := 200
	c := conf.Db.C(params["coll"])

	// parse body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		status = 400
		return status, jsonResponse(err, nil)
	}
	var body bson.M
	err = json.Unmarshal(b, &body)
	if err != nil {
		status = 400
		return status, jsonResponse(err, nil)
	}

	result := map[string]bool{
		"inserted": true,
	}
	err = c.Insert(body)
	if err != nil {
		result["inserted"] = false
		status = 400
	}
	return status, jsonResponse(err, result)
}

func put(req *http.Request, params martini.Params) (int, string) {
	status := 200
	c := conf.Db.C(params["coll"])
	response := map[string]int{}
	_id, ok := parseId(params["_id"])
	// not ok if _id == ""
	if !ok {
		status = 400
		response["updated"] = 0
		return status, jsonResponse(errors.New("invalid _id"), nil)
	}

	count, err := c.Find(bson.M{"_id": _id}).Count()
	response["updated"] = count
	if count == 0 {
		status = 404
		return status, jsonResponse(errors.New("not found"), nil)
	}

	// parse body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		status = 400
		return status, jsonResponse(err, nil)
	}
	var body interface{}
	err = json.Unmarshal(b, &body)
	if err != nil {
		status = 400
		return status, jsonResponse(err, nil)
	}

	v, _ := body.(map[string]interface{})
	delete(v, "_id")

	err = c.UpdateId(_id, v)
	if err != nil {
		status = 400
		return status, jsonResponse(err, nil)
	}

	return status, jsonResponse(err, response)
}

func del(req *http.Request, params martini.Params) (int, string) {
	status := 200
	c := conf.Db.C(params["coll"])
	_id, ok := parseId(params["_id"])
	response := map[string]int{"removed": 1}
	if !ok {
		response["removed"] = 0
		status = 400
		return status, jsonResponse(errors.New("invalid _id"), nil)
	}

	err := c.RemoveId(_id)
	if err != nil {
		status = 400
		if err.Error() == "not found" {
			status = 404
			response["removed"] = 0
		}
	}
	return status, jsonResponse(err, response)
}

func Serve(c *Config) martini.Handler {
	if c == nil {
		panic("rest: please specify a config!")
	}

	conf = c

	r := martini.NewRouter()
	// remove NotFound from bundle
	r.NotFound(make([]martini.Handler, 0)...)

	// TODO: middleware
	r.Get(c.Prefix+"/:coll", get)
	r.Get(c.Prefix+"/:coll/:_id", getId)
	r.Post(c.Prefix+"/:coll", post)
	r.Put(c.Prefix+"/:coll/:_id", put)
	r.Delete(c.Prefix+"/:coll/:_id", del)

	return r.Handle
}
