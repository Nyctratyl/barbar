package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	iconCPU           = "\uf44d"
	iconDateTime      = "\uf246"
	iconMemory        = "\uf289"
	iconNetRX         = "\uf145"
	iconNetTX         = "\uf157"
	iconPowerBattery  = "\uf177"
	iconPowerCharging = "\uf17b"
	iconVolume        = "\uf50a"
	iconVolumeMuted   = "\uf66e"
	fieldSeparator = " | "
)

const (
	SCRIPTS_PATH = "/home/jacob/repos/scripts/"
)

var (
	unmutedLine = regexp.MustCompile("^[[:blank:]]*Mute: no$")
	volumeLine = regexp.MustCompile("^[[:blank:]]*Volume: ")
	channelVolume = regexp.MustCompile("[[:digit:]]+%")
	currentKpi int
	currentCompany int
)

// updatePower reads the current battery and power plug status
func GetPower() string {
	const powerSupply = "/sys/class/power_supply/"
	var enFull, enNow int = 0, 0
	readval := func(name, field string) int {
		var path = powerSupply + name + "/"
		var file []byte
		if tmp, err := ioutil.ReadFile(path + "energy_" + field); err == nil {
			file = tmp
		} else if tmp, err := ioutil.ReadFile(path + "charge_" + field); err == nil {
			file = tmp
		} else {
			return 0
		}

		if ret, err := strconv.Atoi(strings.TrimSpace(string(file))); err == nil {
			return ret
		}
		return 0
	}

	battery := "BAT0"
	enFull += readval(battery, "full")
	enNow += readval(battery, "now")
	var status, _ = ioutil.ReadFile(powerSupply+battery+"/status")
	var statusStr = strings.TrimSpace(string(status))
	p := enNow * 100 / enFull

	return fmt.Sprintf("Battery: %3d (%s)", p, statusStr)
}

func GetBorsdata() string {
	body, err := ioutil.ReadFile("/home/gasha/kod/bar_bar/borsdata_config.json")
	if err != nil {
		return "failed to read config for kpi"
	}
	var kpiConfig map[string][]string
	err = json.Unmarshal(body, &kpiConfig)
	kpiCompanies, ok := kpiConfig["kpiCompanies"]
	if !ok {
		return "failed to read kpi companies"
	}
	kpiNames, ok := kpiConfig["kpiNames"]
	if !ok {
		return "failed to read kpi names"
	}
	if currentKpi >= len(kpiNames) {
		currentKpi = 0
		currentCompany += 1
	}
	if currentCompany >= len(kpiCompanies) {
		currentCompany = 0
	}
	cmpny := kpiCompanies[currentCompany]
	kpi := kpiNames[currentKpi]
	currentKpi += 1
	cmd := exec.Command(SCRIPTS_PATH + "borsdata_fetch.sh", cmpny, kpi)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return "ERR" 
	}
	cmpny = strings.ReplaceAll(cmpny, "-", " ")
	cmpny = strings.Title(cmpny)
	return fmt.Sprintf("%s: %s %s", cmpny, kpi, string(out))
}

// updateVolume reads the volume from pulseaudio
func GetVolume() string {
	cmd := exec.Command("pactl", "list", "sinks")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return "ERR" + iconVolume
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	chanCount := 0
	volSum := 0
	mute := " (Muted)"
	for scanner.Scan() {
		line := scanner.Text()
		if unmutedLine.MatchString(line) {
			mute = ""
		}
		if !volumeLine.MatchString(line) {
			continue
		}
		m := channelVolume.FindAllString(line, -1)
		for _, c := range m {
			var v int
			if _, err := fmt.Sscanf(c, "%d%%", &v); err == nil {
				chanCount++
				volSum += v
			}
		}
	}
	if err := scanner.Err(); err != nil || chanCount == 0 {
		return "ERR" + iconVolume
	}

	p := volSum/chanCount
	return fmt.Sprintf("Volume: %d%s", p, mute)
}

func GetDisk() string {
	cmd := exec.Command(SCRIPTS_PATH + "disk.fish")
	out, _ := cmd.Output()
	outString := strings.TrimSpace(string(out))
	return fmt.Sprintf("%s", outString)
}

func GetWifi() string {
	cmd := exec.Command(SCRIPTS_PATH + "wifi.sh")
	out, _ := cmd.Output()
	outString := strings.TrimSpace(string(out))
	if len(outString) > 0 {
		return fmt.Sprintf("%s", outString)
	} else {
		return fmt.Sprintf("WIFI: Disconnected")
	}
}

func GetMusic() string {
	metaCmd := exec.Command(SCRIPTS_PATH + "music.fish")
	meta, err := metaCmd.Output()
	separator := " - "
	if err != nil {
		return fmt.Sprintf(separator)
	}
	metaString := string(meta)
	if len(metaString) == 0 {
		return fmt.Sprintf(separator)
	}
	if metaString == "No players found" {
		return fmt.Sprintf(separator)
	}
	split := strings.Split(metaString, ";")
	player := strings.TrimSpace(split[0])
	artist := strings.TrimSpace(split[2])
	title := strings.TrimSpace(split[1])
	add := ""

	// Special case if-statement that should only execute if Spotify is streaming music, and
	// thus not giving any metadata as response. If this executes in another situation, bugfix.
	if player == "spotify" && len(title) + len(artist) == 0 {
		sonosCmd := exec.Command(SCRIPTS_PATH + "sonos.fish")
		sonos, _ := sonosCmd.Output()
		sonosString := strings.TrimSpace(html.UnescapeString(string(sonos)))
		split := strings.Split(sonosString, ", a song by ")
		add = " (streaming)"
		if len(split) > 1 {
			artist = split[1]
			title = split[0]
		}
	}
	if len(artist) > 20 {
		artist = artist[0:20] + "..."
	}
	if len(title) > 20 {
		title = title[0:20] + "..."
	}
	playerString := "(" + player + ") "
	outString := playerString + title + separator + artist + add
	statusCmd := exec.Command(SCRIPTS_PATH + "music_status.fish")
	status, _ := statusCmd.Output()
	statusString := strings.TrimSpace(string(status))
	if statusString == "Paused" {
		outString += " (Paused)"
	}
	return fmt.Sprintf("%s", outString)
}

