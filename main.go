package main

import (
	_ "embed"
	"log"
	"os"
	"os/exec"
	"text/template"
)

var configs = []Sensor{
	{Name: "storage", IpAddress: "10.0.64.10"},
}

type Sensor struct {
	Name string
	IpAddress string
	Secrets Secrets
}

type Secrets struct {
	EspAPIKey string
	OtaPassword string
	WifiSSID string
	WifiPassword string
}

//go:embed template.yaml
var espHomeTemplate string
func main() {
	// instantiate template
	var tmpl = template.Must(template.New("espconf").Parse(espHomeTemplate))

	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <deviceName> </dev/usbttyS0,COM3,192.168.1.100,foo.local>\n", os.Args[0])
	}
	deviceName := os.Args[1]

	// pick device from config
	var deviceConfig *Sensor
	for _, dc := range configs {
		if dc.Name == deviceName {
			dc.Secrets = secrets
			deviceConfig = &dc
		}
	}
	if deviceConfig == nil {
		log.Fatal("No config for ", deviceName)
	}

	// make temporary file
	f, err := os.CreateTemp("", deviceName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name()) // clean up

	// calculate config for device
	err = tmpl.Execute(f, *deviceConfig)
	if err != nil {
		log.Fatal("interpreting config: ", err)
	}
	f.Close()

	cmd := exec.Command("esphome", "upload", "--device", f.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Println(string(output))
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Fatal("esphome error: ", string(exitErr.Stderr))
		} else {
			log.Fatal("failed to execute esphome: ", exitErr.Stderr)
		}
	}		
}
