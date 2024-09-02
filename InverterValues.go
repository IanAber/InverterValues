package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var sources []string

func init() {
	var sourcesString string
	// open log file
	logFile, err := os.OpenFile("/var/log/InverterValues", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	flag.StringVar(&sourcesString, "sources", "firefly", "Comma separated list of sources")
	flag.Parse()

	sources = strings.Split(sourcesString, ",")
	log.Println("Sources = ", sources)
}

type inverterValues struct {
	Source    string  `json:"source"`
	IBatt     float64 `json:"iBatt"`
	VBatt     float64 `json:"vBatt"`
	SOC       float64 `json:"soc"`
	Frequency float64 `json:"frequency"`
	Solar     uint64  `json:"solar"`
}

type aberhomeValues struct {
	Inverter inverterValues `json:"inverter"`
}

func getInverterValues() *inverterValues {
	if resp, err := http.Get("http://aberhome1.home:8080"); err != nil {
		log.Print(err)
		return nil
	} else {
		defer func() {
			if resp.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		bodyBytes, _ := io.ReadAll(resp.Body)

		var data aberhomeValues
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			log.Print(err)
			return nil
		}
		data.Inverter.IBatt = 0 - data.Inverter.IBatt
		data.Inverter.Source = "firefly"
		return &data.Inverter
	}
}

type solarValues struct {
	A uint64 `json:"A"`
	B uint64 `json:"B"`
	C uint64 `json:"C"`
	D uint64 `json:"D"`
	E uint64 `json:"E"`
	F uint64 `json:"F"`
	G uint64 `json:"G"`
	H uint64 `json:"H"`
	I uint64 `json:"I"`
	J uint64 `json:"J"`
	K uint64 `json:"K"`
}

type realTimeData struct {
	Solar solarValues `json:"solar"`
}

func getSolar() (uint64, error) {
	if resp, err := http.Get("http://aberhome1.home:8080/realtime/getdata"); err != nil {
		return 0, err
	} else {
		defer func() {
			if resp.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		bodyBytes, _ := io.ReadAll(resp.Body)

		var data realTimeData
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			return 0, err
		}
		solarTotal := data.Solar.A + data.Solar.B + data.Solar.C + data.Solar.D + data.Solar.E + data.Solar.F + data.Solar.G + data.Solar.H + data.Solar.I + data.Solar.J + data.Solar.K
		return solarTotal, nil
	}
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		if data := getInverterValues(); data != nil {
			if solar, err := getSolar(); err != nil {
				log.Print(err)
			} else {
				data.Solar = solar
			}
			for _, source := range sources {
				data.Source = source
				if jsonReq, err := json.Marshal(data); err != nil {
					log.Print(err)
				} else {
					if req, err := http.NewRequest("POST", "http://firefly.home:8080/recordPower", bytes.NewBuffer(jsonReq)); err != nil {
						log.Println(err)
					} else {
						req.Header.Add("authorization", "9cb252bf-949f-461b-a2ed-f3d519fd5a2f")
						_, err = http.DefaultClient.Do(req)
					}
				}
			}
		} else {
			fmt.Println("Nothing returned")
		}
		time.Sleep(time.Second)
	}
}
