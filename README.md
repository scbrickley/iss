# ISS tracker

*A data scraper that writes the longitude and latitude of the International Space Station to an InfluxDB instance.*

## Installation

Assuming you have the Go programming language installed on your machine, you can run the following commands to install the `iss` executable on your machine:

- `git clone git@github.com:scbrickley/iss.git`
- `cd iss`
- `go build && go install`

## Usage

Here's an example of how to use the `iss` command line tool:

`iss -url=localhost:9999 -auth=../path/to/auth/file -org=<name-of-org> -bucket=<name-of-bucket>`

Once you run that command, `iss` will start querying the data about once per second. It will then format the data as line protocol and save it to a buffer. Once it has collected 100 data points, it will make the `POST` request to the write API of the InfluxDB instance you've specified, clear its buffer, and start collecting data again.


## Drawing the data to a map

If you have geo-temporal features available on your InfluxDB instance, then you can draw these data points to a world map by selecting the `Map` visualization, selecting `Circle map` as the visualization type, and creating a query with the following Flux script:

```
import "experimental/geo"
from(bucket: "ISS")
  |> range(start: -2h)
  |> filter(fn: (r) => r["_measurement"] == "iss_position")
  |> geo.shapeData(latField: "latitude", lonField: "longitude", level: 10)
```
