package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	volumeString string
	musicString  string
	wifiString   string
	powerString  string
	diskString   string
	borsdataString string
)

type Config struct {
  Port int `json:"port"`
  Modules []string `json:"modules"`
}

func updateLoop(getter func() string, target *string, frequency int) {
	go func() {
		for {
			*target = getter()
			var now = time.Now()
			time.Sleep(now.Truncate(time.Second).Add(time.Second * time.Duration(frequency)).Sub(now))
		}
	}()
}

func draw() {
	now := time.Now()
	status := []string{""}
	statusRaw := []string{
		borsdataString,
		volumeString,
		musicString,
		wifiString,
		powerString,
		diskString,
		now.Local().Format("15:04:05 (02-01-2006)"),
	}
	for _, v := range statusRaw {
  	if v != "" {
    	status = append(status, v)
  	}
	}
	s := strings.Join(status, fieldSeparator)
	fmt.Println(s)
}
func forceUpdateHandler(writer http.ResponseWriter, req *http.Request) {
	go func() {
		switch req.URL.Query().Get("d") {
		case "volume":
			volumeString = GetVolume()
		case "music":
			musicString = GetMusic()
		case "wifi":
			wifiString = GetWifi()
		case "power":
			powerString = GetPower()
		case "disk":
			diskString = GetDisk()
		}
		draw()
	}()
	writer.WriteHeader(http.StatusOK)
}

func main() {

	configFile, err := os.ReadFile("config.json")
	if err != nil {
  	fmt.Println("failed to read config, falling back to default")
		configFile, err = os.ReadFile("config.default.json")
	}
	if err != nil {
  	fmt.Println("failed to read default config")
  	panic(0)
	}
	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
  	fmt.Println("failed to marshal config")
  	panic(0)
	}

	for _, v := range config.Modules {
  	switch v {
    	case "music":
				updateLoop(GetMusic, &musicString, 10)
			case "volume":
        updateLoop(GetVolume, &volumeString, 60)
			case "wifi":
        updateLoop(GetWifi, &wifiString, 10)
			case "power":
        updateLoop(GetPower, &powerString, 10)
			case "disk":
        updateLoop(GetDisk, &diskString, 30)
			case "borsdata":
        updateLoop(GetBorsdata, &borsdataString, 600)
  	}
	}

	http.HandleFunc("/forceUpdate", forceUpdateHandler)
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	}()

	for {
		draw()
		var now = time.Now()
		time.Sleep(now.Truncate(time.Second).Add(time.Second).Sub(now))
	}
}
