package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ORG        = "Myself"
	BUCKET     = "ISS"
	TOKEN_FILE = "/Users/seanbrickley/.tok/ossinflux"
)

type issPos struct {
	Lat  string `json:"latitude"`
	Long string `json:"longitude"`
}

type issInfo struct {
	Timestamp int    `json:"timestamp"`
	Pos       issPos `json:"iss_position"`
}

func check(e error) {
	if e != nil {
		log.Fatalf("Error: %s", e)
	}
}

func main() {
	// New Buffer to write line protocol data to
	buf := bytes.NewBufferString("")

	// Declare incrementer outside the loop. We want to run the loop
	// indefinitely, but send the data in batches, to reduce the number 
	// of requests. The iterator will be reset periodically when the data
	// is flushed.
	i := 0

	for {
		issData, err := issData() // Pull data from public API
		log.Info("Querying data from ISS API")
		if err != nil {
			log.Fatalf("Error fetching data: %s\n", err)
		}

		lp := issData.toLineProtocol()
		log.Infof("Data converted to line protocol:\n%s", lp)

		log.Info("Writing to buffer")
		buf.WriteString(lp)
		i++
		log.Infof("Data points in buffer: %d", i)

		// Set batch size here
		if i >= 100 {
			i = 0
			send(buf)
		}

		time.Sleep(time.Second)
	}
}

func send(buf *bytes.Buffer) {
	log.Infof("Writing data to InfluxDB API v2")

	client := &http.Client{}

	url := fmt.Sprintf(
		"http://localhost:9999/api/v2/write?org=%s&bucket=%s&precision=s",
		ORG,
		BUCKET,
	)

	req, err := http.NewRequest("POST", url, buf)
	check(err)

	// Set the Authentication header
	req.Header.Set("Authorization", "Token " + tok())
	_, err = client.Do(req)
	check(err)

	log.Info("Resetting buffer")
	buf.Reset()
}

func issData() (issInfo, error) {
	// issInfo struct to return if something goes wrong,
	// since we can't just return `nil`
	errStruct := issInfo {
		Timestamp: 0,
		Pos: issPos {
			Lat:  "0.0",
			Long: "0.0",
		},
	}

	resp, err := http.Get("http://api.open-notify.org/iss-now.json")
	if err != nil {
		return errStruct, err
	}

	info, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errStruct, err
	}

	var position issInfo

	err = json.Unmarshal(info, &position)
	if err != nil {
		return errStruct, err
	}

	return position, nil
}

func (i issInfo) toLineProtocol() string {
	return fmt.Sprintf("iss_position latitude=%s,longitude=%s %d\n",
		i.Pos.Lat,
		i.Pos.Long,
		i.Timestamp,
	)
}

func tok() string {
	token, err := ioutil.ReadFile(TOKEN_FILE)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(token))
}
