package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	org    = flag.String("org", "Home", "Name of the organization that owns the bucket")
	bucket = flag.String("bucket", "ISS", "Name of the bucket you want to write the data to")
	url    = flag.String("url", "localhost:9999/", "Base URL for your InfluxDB instance")
	auth   = flag.String("auth", "", "Path to an plain text file that holds your auth token (and nothing else)")
)

func init() {
	flag.Parse()
}

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

	writeAPI := fmt.Sprintf(
		"http://%s/api/v2/write?org=%s&bucket=%s&precision=s",
		*url,
		*org,
		*bucket,
	)

	req, err := http.NewRequest("POST", writeAPI, buf)
	check(err)

	// Set the Authentication header
	req.Header.Set("Authorization", "Token "+tok())
	_, err = client.Do(req)
	check(err)

	log.Info("Resetting buffer")
	buf.Reset()
}

func issData() (*issInfo, error) {
	resp, err := http.Get("http://api.open-notify.org/iss-now.json")
	if err != nil {
		return nil, err
	}

	info, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var position issInfo
	err = json.Unmarshal(info, &position)
	if err != nil {
		return nil, err
	}

	return &position, nil
}

func (i issInfo) toLineProtocol() string {
	return fmt.Sprintf("iss_position latitude=%s,longitude=%s %d\n",
		i.Pos.Lat,
		i.Pos.Long,
		i.Timestamp,
	)
}

func tok() string {
	token, err := ioutil.ReadFile(*auth)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(token))
}
