package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"
)

var configs = []Sensor{
	{Name: "storage", IpAddress: "10.0.64.10"},
}

type Sensor struct {
	Name      string
	IpAddress string
	Secrets   Secrets
}

type Secrets struct {
	EspAPIKey    string
	OtaPassword  string
	WifiSSID     string
	WifiPassword string
}

//go:embed template.yaml
var espHomeTemplate string

func main() {
	// instantiate template
	var tmpl = template.Must(template.New("espconf").Parse(espHomeTemplate))

	for _, deviceConfig := range configs {
		deviceConfig.Secrets = secrets

		// write config file
		filename := fmt.Sprintf("%s.yaml", deviceConfig.Name)
		f, err := os.Create(path.Join("configs", filename))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// write config for device
		err = tmpl.Execute(f, deviceConfig)
		if err != nil {
			log.Fatal("interpreting config: ", err)
		}
	}
}
