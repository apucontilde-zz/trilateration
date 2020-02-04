package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

type InputLine struct {
	RaspberryPiId   int64
	LastTime        string  `json:"kismet.device.base.last_time"`
	Macaddr         string  `json:"kismet.device.base.macaddr"`
	Distance        float64 `json:"distancia_senal_mediana_f2"`
	SignalIntensity float64 `json:"minute_vec_signal_med"`
}

type OutputLine struct {
	Time      string     `csv:"time"`
	Macaddr   string     `csv:"macaddr"`
	Distances [3]float64 `csv:"distances"`
	X         float64    `csv:"x"`
	Y         float64    `csv:"y"`
}

type RaspberryPi struct {
	RaspberryPiId int64   `json:"raspberrypi_id"`
	InputUrl      string  `json:"input_url"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
}

type Config struct {
	RaspberryPis []RaspberryPi `json:"raspberrypis"`
	OutputUrl    string        `json:"output_url"`
}

const CONFIG_FILENAME = "trilateration_config.json"
const OUTFILE_NAME = "results.csv"
const PRINT_DEBUG = false
const SAVE_STATS = false

var processing_channels map[string]chan InputLine
var RB_X_POSITIONS [3]float64
var RB_Y_POSITIONS [3]float64

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func loadConfig(configFilename string) (config Config, err error) {
	configFile, err := os.Open(CONFIG_FILENAME)
	if err != nil {
		return config, errors.New("Couldn't open the configuration file")
	}
	defer configFile.Close()
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return config, errors.New("Couldn't decode the configuration file")
	}

	return config, nil
}

func main() {

	// read RaspberryPi info from configfile.
	config, err := loadConfig(CONFIG_FILENAME)
	checkError("loadConfig failed", err)
	log.Printf("Loaded config file: \n%v", config)
	// create getData channel to get data from pis
	input_channel := make(chan InputLine, 1000)
	for idx, rbPi := range config.RaspberryPis {
		RB_X_POSITIONS[idx] = rbPi.X
		RB_Y_POSITIONS[idx] = rbPi.Y
		go GetData(rbPi, input_channel)
	}

	var detected_addresses, outfile *os.File
	if SAVE_STATS {
		detected_addresses, err := os.OpenFile("detected_addresses.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		checkError("could not open outfile", err)
		defer detected_addresses.Close()

		outfile, err := os.OpenFile(OUTFILE_NAME, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		checkError("could not open outfile", err)
		defer outfile.Close()
	}

	// create macaddress:channel to process each maccaddress concurrently
	processing_channels := make(map[string]chan InputLine)
	output_channel := make(chan OutputLine)
	go func() {
		for line := range input_channel {
			outstring := fmt.Sprintf("\"%s\",\"%s\",%d,%f,%f,\n", line.LastTime, line.Macaddr, line.RaspberryPiId, line.Distance, line.SignalIntensity)
			if SAVE_STATS {
				detected_addresses.WriteString(outstring)
			}
			if PRINT_DEBUG {
				log.Println(outstring)
			}
			if line.Macaddr != "" {
				_, ok := processing_channels[line.Macaddr]
				if !ok {
					processing_channels[line.Macaddr] = make(chan InputLine, 1000)
					go trilaterate(line.Macaddr, processing_channels[line.Macaddr], output_channel)
				}
				processing_channels[line.Macaddr] <- line
			} else {
				if PRINT_DEBUG {
					log.Println("no macaddr in line, continuing")
				}
			}
		}
	}()

	for line := range output_channel {
		go SendData(line, config.OutputUrl)

		outstring := fmt.Sprintf("\"%s\",\"%s\",%f,%f,%f,%f,%f,\n", line.Macaddr, line.Time, line.Distances[0], line.Distances[1], line.Distances[2], line.X, line.Y)
		if PRINT_DEBUG {
			log.Println(outstring)
		}
		if SAVE_STATS {
			_, err = outfile.WriteString(outstring)
		}
		checkError("could not write to outfile", err)

	}

}

func GetData(rbPi RaspberryPi, input_channel chan InputLine) {
	var lastline *InputLine
	for {
		var url = rbPi.InputUrl
		params := "?timestamp="
		if lastline != nil {
			params += lastline.LastTime
			url += params
		}
		if PRINT_DEBUG {
			log.Println("Getting data from RBpi", rbPi.RaspberryPiId, url)
		}
		response, err := http.Get(url)
		if err != nil {
			log.Println("Couldn't get data from pi", rbPi.RaspberryPiId, "\n", err)
		} else {
			defer response.Body.Close()
			lastline = ReadJSON(response.Body, input_channel, rbPi.RaspberryPiId)
		}
		time.Sleep(2 * time.Second)

	}
}

func SendData(data OutputLine, outputUrl string) {
	jsonStr, err := json.Marshal(data)
	checkError("failed convert output to json", err)
	req, err := http.NewRequest("POST", outputUrl, bytes.NewBuffer(jsonStr))
	checkError("failed create http request", err)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	checkError("failed http POST", err)
	defer resp.Body.Close()
	log.Println("Sent trilateration to server", resp.Status)
}

func ReadJSON(jsonStream io.Reader, input_channel chan InputLine, RaspberryPiId int64) (lastline *InputLine) {
	dec := json.NewDecoder(jsonStream)
	// read open bracket
	_, err := dec.Token()
	if err != nil {
		log.Panic("ReadJSON", err)
	}
	// while the array contains values
	for dec.More() {
		var d InputLine
		// decode an array value (Message)
		err := dec.Decode(&d)
		if err != nil {
			log.Panic("ReadJSON", err)
		}

		d.RaspberryPiId = RaspberryPiId
		// if d.Macaddr == "00:08:22:27:75:8A" || d.Macaddr == "A4:D9:31:CC:9B:62" || d.Macaddr == "C2:4D:6C:92:58:08" || d.Macaddr == "90:81:2A:34:75:E7" {
		if PRINT_DEBUG {
			log.Printf("%v", d)
		}
		input_channel <- d
		lastline = &d
		// }

	}
	// read closing bracket
	_, err = dec.Token()
	if err != nil {
		log.Panic("ReadJSON", err)
	}

	return lastline
}

func trilaterate(macaddr string, lines chan InputLine, output chan OutputLine) {
	log.Println("Starting trilateration thread for macaddr", macaddr)
	var r = [3]float64{0, 0, 0}
	for line := range lines {
		r[line.RaspberryPiId-1] = line.Distance
		if PRINT_DEBUG {
			log.Println(macaddr, r)
		}
		if r[0] > 0 && r[1] > 0 && r[2] > 0 {
			X, Y := trilateration(r, RB_X_POSITIONS, RB_Y_POSITIONS)
			log.Println("trilateration of macaddr", macaddr, r, ": ", X, ",", Y)
			output <- OutputLine{line.LastTime, macaddr, r, X, Y}
			r = [3]float64{0, 0, 0}
		}
	}
}

func trilateration(r [3]float64, x [3]float64, y [3]float64) (X float64, Y float64) {
	A := -2*x[0] + 2*x[1]
	B := -2*y[0] + 2*y[1]
	C := math.Pow(r[0], 2) - math.Pow(r[1], 2) - math.Pow(x[0], 2) + math.Pow(x[1], 2) - math.Pow(y[0], 2) + math.Pow(y[1], 2)

	D := -2*x[1] + 2*x[2]
	E := -2*y[1] + 2*y[2]
	F := math.Pow(r[1], 2) - math.Pow(r[2], 2) - math.Pow(x[1], 2) + math.Pow(x[2], 2) - math.Pow(y[1], 2) + math.Pow(y[2], 2)

	denominator := (A*E - B*D)
	X = (C*E - F*B) / denominator
	Y = (A*F - C*D) / denominator
	return X, Y
}
