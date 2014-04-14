## REST 
Simple [REST](http://en.wikipedia.org/wiki/Representational_state_transfer) interface over MongoDB, as middlware for [martini](https://github.com/g-martini/martini) framework. Useful to create single page applications, REST style based.

#### Usage:

```go
package main

import (
  "github.com/go-martini/martini"
  "labix.org/v2/mgo"
  "github.com/olebedev/rest"
)

func access(res http.ResponseWriter, req *http.Request){
  // Your data access logic here
}

func main() {
  session, err := mgo.Dial("localhost")
  if err != nil {
    panic(err)
  }
  defer session.Close()
  session.SetMode(mgo.Monotonic, true)
  db := session.DB("test")

  m := martini.Classic()
  
  m.Group("/api/v1", rest.Rest(rest.Config{
    Db:           db, 
    ResonseField: "data", // optional
    // Use integer autoincrement for _id instead of mongodb auto generated hash, default false. optional
    // Autoincrement: true, 
  }, access))

  m.Run()
}
```

Now you can send HTTP requests to `http://localhost:3000/api/v1/example_collection`.  
Available `GET` parameters:  

- query - JSON mongodb [query](http://www.mongodb.org/display/DOCS/Querying) statement
- limit - `int`
- skip - `int`
- sort - `string`, [more detail](http://godoc.org/labix.org/v2/mgo#Query.Sort)
- count - `bool`
- select - JSON mongodb [select](http://www.mongodb.org/display/DOCS/Retrieving+a+Subset+of+Fields) statement

#### Examples:

Let's create something simple.
```bash
$ curl -X POST http://localhost:5000/api/v1/test -s \
  -H "Accept: application/json" \
  -H "Content-type: application/json" \
  -d '{"hello":"world"}'

{
  "data": { // ResonseField name
    "_id":"52d382ae367e0ee611626bf1"
    "hello":"world"
  }
}
$ // ... and so on
```

To get full collection use this command:
```bash
$ curl http://localhost:5000/api/v1/test
{
  "data": [
    {
      "hello": "world",
      "_id": "52d382ae367e0ee611626bf0"
    },
    {
      "hello": "world 2",
      "_id": "52d382b0367e0ee611626bf1"
    },
    {
      "hello": "world 3",
      "_id": "52d382b4367e0ee611626bf2"
    }
  ]
}
```

Also available PUT(by id) & DELETE(by id) methods.
