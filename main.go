package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"net/url"

	"github.com/kpawlik/geojson"
	"github.com/namsral/flag"
)

type (
	Worker struct {
		*sync.WaitGroup
		Keys     []string
		Endpoint string
		Jobs     chan string
		results  chan map[string]interface{}
		done     chan struct{}
	}
)

var (
	keys, geocoder string
)

func init() {
	flag.StringVar(&keys, "keys", "", "keys for life")
	// flag.StringVar(&geocoder, "geocoder", "", "geocoder endpoint")
	flag.Parse()

	geocoder = "http://maps.googleapis.com/maps/api/geocode/json?address="
}

func main() {
	if keys == "" || geocoder == "" {
		fmt.Println("please supply both `keys` and `geocoder`")
		flag.Usage()
		os.Exit(1)
	}

	fields := strings.Split(keys, ",")

	worker := NewWorker(fields, geocoder)
	for i := 0; i < runtime.NumCPU(); i++ {
		worker.Add(1)
		go worker.Background()
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		worker.Put(scanner.Text())
	}
	worker.Close()
	worker.Wait()
	fmt.Println("done!")
}

func NewWorker(keys []string, geocoder string) *Worker {
	worker := &Worker{
		WaitGroup: &sync.WaitGroup{},
		Keys:      keys,
		Endpoint:  geocoder,
		Jobs:      make(chan string, 100),
		results:   make(chan map[string]interface{}, 100),
		done:      make(chan struct{}, 1),
	}
	go worker.Write()
	return worker
}

func (w *Worker) Background() {
	var parsed, geocoded map[string]interface{}
	client := &http.Client{}
	for {
		line, more := <-w.Jobs
		if !more {
			break
		}

		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			log.Printf("input: %s, err: %v \n", line, err)
			continue
		}

		var previous string
		for _, key := range w.Keys {
			if previous == "" {
				previous = fmt.Sprintf("%s", parsed[key])
			} else {
				previous = fmt.Sprintf("%s, %s", previous, parsed[key])
			}
		}

		previous = url.QueryEscape(previous)
		u := fmt.Sprintf("%s%s", w.Endpoint, previous)
		fmt.Printf("requesting %s...\n", u)

		resp, err := client.Get(u)
		if err != nil {
			log.Printf("input: %s, err: %v \n", u, err)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			log.Printf("expecting 200, got %d, input: %s", resp.StatusCode, u)
			continue
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			log.Println(err)
			continue
		}

		if err := json.Unmarshal(b, &geocoded); err != nil {
			log.Println(err)
			continue
		}

		if geocoded["results"] != nil {
			if len(geocoded["results"].([]interface{})) > 0 {
				geom := geocoded["results"].([]interface{})[0].(map[string]interface{})["geometry"]
				if geom != nil {
					if geom.(map[string]interface{})["location"] != nil {
						location := geom.(map[string]interface{})["location"].(map[string]interface{})
						parsed["_latitude"] = location["lat"]
						parsed["_longitude"] = location["lng"]
						w.results <- parsed
					}
				}
			}
		}
	}
	w.Done()
}

func (w *Worker) Put(line string) {
	w.Jobs <- line
}

func (w *Worker) Write() {
	features := make([]*geojson.Feature, 0)
	for {
		target, more := <-w.results
		if !more {
			break
		}

		if target["_latitude"] != nil && target["_longitude"] != nil {
			lat := target["_latitude"].(float64)
			lng := target["_longitude"].(float64)
			point := geojson.NewPoint(geojson.Coordinate{
				geojson.CoordType(lng),
				geojson.CoordType(lat),
			})
			f := geojson.NewFeature(point, target, nil)
			features = append(features, f)
		}
	}

	fc := geojson.NewFeatureCollection(features)
	b, err := json.Marshal(fc)
	if err != nil {
		log.Println(err)
	} else {
		if err := ioutil.WriteFile(fmt.Sprintf("%d.geojson", time.Now().Unix()), b, 0644); err != nil {
			log.Println(err)
		}
	}

	w.done <- struct{}{}
}

func (w *Worker) Close() {
	close(w.Jobs)
}

func (w *Worker) Wait() {
	w.WaitGroup.Wait()
	close(w.results)
	<-w.done
}
