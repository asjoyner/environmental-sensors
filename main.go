package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
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

type ZoneHeader struct {
	Origin string
	Serial string
}

//go:embed template.yaml
var espHomeTemplate string

//go:embed dns-header.tmpl
var dnsHeader string

// This generates a new serial for each hour, which is enough for the task
func dnsSerial() string {
	// Get the current time.
	now := time.Now()

	// Format the date part as YYYYMMDD.
	datePart := now.Format("20060102")

	// Get the current hour as an integer (0-23).
	hourInt := now.Hour()

	// Convert the hour integer to a string and zero-pad to two digits.
	// For example, 9 becomes "09".
	hourStr := fmt.Sprintf("%02d", hourInt)

	// Combine the date and hour to form the DNS serial.
	return datePart + hourStr
}

func main() {
	// instantiate ESPHome template
	var espTmpl = template.Must(template.New("espconf").Parse(espHomeTemplate))

	// instantiate DNS header template
	var dnsTmpl = template.Must(template.New("dnsHeader").Parse(dnsHeader))

	// create UniFi DHCP lease CSV
	leasesFileHandle, err := os.Create("unifi-dhcp.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer leasesFileHandle.Close()

	// create DNS zone files
	cfg := ZoneHeader{Serial: dnsSerial()}
	aRecordFileHandle, err := os.Create("db.iot.joyner.ws")
	if err != nil {
		log.Fatal(err)
	}
	defer leasesFileHandle.Close()
	cfg.Origin = "iot.joyner.ws."
	err = dnsTmpl.Execute(aRecordFileHandle, cfg)
	if err != nil {
		log.Fatal("interpreting config: ", err)
	}

	PTRFileHandle, err := os.Create("db.64.0.10")
	if err != nil {
		log.Fatal(err)
	}
	defer leasesFileHandle.Close()
	cfg.Origin = "64.0.10.in-addr.arpa."
	err = dnsTmpl.Execute(PTRFileHandle, cfg)
	if err != nil {
		log.Fatal("interpreting config: ", err)
	}

	for _, dc := range configs {
		dc.Secrets = secrets

		// write config file
		filename := fmt.Sprintf("%s.yaml", dc.Name)
		err := os.MkdirAll("configs", 0700)
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(path.Join("configs", filename))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// write config for device
		err = espTmpl.Execute(f, dc)
		if err != nil {
			log.Fatal("interpreting config: ", err)
		}

		dnsName := fmt.Sprintf("%s.temphum.iot.joyner.ws.", dc.Name)
		// append to leases file
		entry := fmt.Sprintf("%s,%s,%s,,,,\n", dc.MAC, dc.IpAddress, dnsName)
		leasesFileHandle.Write([]byte(entry))

		// append to DNS files
		entry = fmt.Sprintf("%s\tA\t%s\n", dnsName, dc.IpAddress)
		aRecordFileHandle.Write([]byte(entry))
		// TODO: actually parse the IP address with a library, maybe?  :)
		entry = fmt.Sprintf("%s\tPTR\t%s\n", strings.Split(dc.IpAddress, ".")[3], dnsName)
		PTRFileHandle.Write([]byte(entry))
	}
}
