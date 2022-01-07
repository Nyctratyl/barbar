package main

import (
	"fmt"
	"net/http"
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
	status := []string{
		"",
		borsdataString,
		volumeString,
		musicString,
		wifiString,
		powerString,
		diskString,
		now.Local().Format("15:04:05 (02-01-2006)"),
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
	updateLoop(GetVolume, &volumeString, 60)
	// updateLoop(GetMusic, &musicString, 60)
	updateLoop(GetWifi, &wifiString, 10)
	updateLoop(GetPower, &powerString, 10)
	updateLoop(GetDisk, &diskString, 30)
	updateLoop(GetBorsdata, &borsdataString, 600)

	http.HandleFunc("/forceUpdate", forceUpdateHandler)
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	for {
		draw()
		var now = time.Now()
		time.Sleep(now.Truncate(time.Second).Add(time.Second).Sub(now))
	}
}
