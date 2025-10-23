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
	{Name: "storage", MAC: "F0:24:F9:97:B8:40", IpAddress: "10.0.64.10"},
}

type Sensor struct {
	Name      string
	MAC       string
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

	// write UniFi DHCP lease CSV
	leasesFileHandle, err := os.Create("unifi-dhcp.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer leasesFileHandle.Close()

	for _, dc := range configs {
		dc.Secrets = secrets

		// write config file
		filename := fmt.Sprintf("%s.yaml", dc.Name)
		f, err := os.Create(path.Join("configs", filename))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// write config for device
		err = tmpl.Execute(f, dc)
		if err != nil {
			log.Fatal("interpreting config: ", err)
		}

		// append to leases file
		entry := fmt.Sprintf("%s,%s,%s,,,,\n", dc.MAC, dc.IpAddress, dc.Name)
		leasesFileHandle.Write([]byte(entry))
	}
}
