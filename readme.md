## REST 
Simple interface under MongoDB, middlware for [martini](https://github.com/codegangsta/martini) framework.

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

#### TODO
- tests
- attach given middleware for rest handlers
- autoincrement int logic as option
