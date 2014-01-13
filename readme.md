## REST 
Simple [REST](http://en.wikipedia.org/wiki/Representational_state_transfer) interface under MongoDB, as middlware for [martini](https://github.com/codegangsta/martini) framework. Useful to create single page applications, based on RESTful style.

#### Usage:

```go
import (
  "github.com/codegangsta/martini"
  "labix.org/v2/mgo"
  "github.com/olebedev/rest"
)

func main() {
  session, err := mgo.Dial("localhost")
  if err != nil {
    panic(err)
  }
  defer session.Close()
  session.SetMode(mgo.Monotonic, true)
  db := session.DB("test")

  m := martini.Classic()
  
  m.Use(rest.Serve(&rest.Config{
    Prefix: "/api/v1/rest",
    Db:     db, 
    ResonseField: "data", // optional
  }))

  m.Run()
}
```

Now you can send HTTP requests to `http://localhost:3000/api/v1/rest/example_collection`.  
Allowed  collection `GET` parameters:  

- query - JSON mongodb [query](http://www.mongodb.org/display/DOCS/Querying) statement
- limit - `int`
- skip - `int`
- sort - `string`, [more detail](http://godoc.org/labix.org/v2/mgo#Query.Sort)
- count - `int`
- select - JSON mongodb [select](http://www.mongodb.org/display/DOCS/Retrieving+a+Subset+of+Fields) statement

#### TODO
- tests
- attach given middleware for rest handlers
- autoincrement int logic as option
