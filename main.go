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
	{MAC: "F0:24:F9:97:C2:98", IpAddress: "10.0.64.10", Name: "vault"},
	{MAC: "F0:24:F9:97:B8:40", IpAddress: "10.0.64.11", Name: "storage"},
	{MAC: "F0:24:F9:xx:xx:xx", IpAddress: "10.0.64.12", Name: "boycave"},
	{MAC: "F0:24:F9:xx:xx:xx", IpAddress: "10.0.64.13", Name: "theater"},
	{MAC: "f0:24:f9:99:68:00", IpAddress: "10.0.64.14", Name: "flex"},
	{MAC: "F0:24:F9:9A:66:4C", IpAddress: "10.0.64.15", Name: "bath-5"},
	{MAC: "F0:24:F9:97:D0:50", IpAddress: "10.0.64.16", Name: "exercise"},
	{MAC: "10:06:1C:17:37:A8", IpAddress: "10.0.64.17", Name: "mechanical"},
	{MAC: "F0:24:F9:97:BD:A8", IpAddress: "10.0.64.18", Name: "linen"},
	{MAC: "F0:24:F9:97:D6:E4", IpAddress: "10.0.64.19", Name: "lavatory-2"},
	{MAC: "10:06:1C:18:D5:E8", IpAddress: "10.0.64.20", Name: "lower-stairwell"},
	{MAC: "F0:24:F9:97:BE:98", IpAddress: "10.0.64.21", Name: "garage-ne"},
	{MAC: "F0:24:F9:9A:64:64", IpAddress: "10.0.64.22", Name: "garage-se"},
	{MAC: "F0:24:F9:97:D7:64", IpAddress: "10.0.64.23", Name: "everything"},
	{MAC: "F0:24:F9:9A:66:28", IpAddress: "10.0.64.24", Name: "kitchen"},
	{MAC: "F0:24:F9:97:B8:24", IpAddress: "10.0.64.25", Name: "livingroom-west"},
	{MAC: "F0:24:F9:9A:E9:24", IpAddress: "10.0.64.26", Name: "master-bedroom"},
	{MAC: "F0:24:F9:9A:EA:EC", IpAddress: "10.0.64.27", Name: "master-bathroom"},
	{MAC: "F0:24:F9:xx:xx:xx", IpAddress: "10.0.64.28", Name: "master-shower"},
	{MAC: "F0:24:F9:xx:xx:xx", IpAddress: "10.0.64.29", Name: "master-closet"},
	{MAC: "F0:24:F9:9A:EA:84", IpAddress: "10.0.64.30", Name: "lavatory"},
	{MAC: "F0:24:F9:97:C1:F8", IpAddress: "10.0.64.31", Name: "bath-2"},
	{MAC: "F0:24:F9:9A:EC:E8", IpAddress: "10.0.64.32", Name: "guest-bedroom"},
	{MAC: "F0:24:F9:9A:EE:4C", IpAddress: "10.0.64.33", Name: "livingroom-east"},
	{MAC: "F0:24:F9:97:D1:78", IpAddress: "10.0.64.34", Name: "mid-stairwell"},
	{MAC: "F0:24:F9:97:BC:28", IpAddress: "10.0.64.35", Name: "bath-3"},
	{MAC: "F0:24:F9:97:D1:BC", IpAddress: "10.0.64.36", Name: "ethan"},
	{MAC: "F0:24:F9:9A:ED:10", IpAddress: "10.0.64.37", Name: "crafts"},
	{MAC: "F0:24:F9:97:D0:04", IpAddress: "10.0.64.38", Name: "aaron"},
	{MAC: "F0:24:F9:95:20:AC", IpAddress: "10.0.64.39", Name: "bath-4"},
	{MAC: "F0:24:F9:99:6A:54", IpAddress: "10.0.64.40", Name: "upper-stairwell"},
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

	// create ISC DHCP lease config
	dhcpdconfFileHandle, err := os.Create("dhcpd-iot.conf")
	if err != nil {
		log.Fatal(err)
	}
	defer dhcpdconfFileHandle.Close()

	// create DNS zone files
	cfg := ZoneHeader{Serial: dnsSerial()}
	aRecordFileHandle, err := os.Create("db.iot.joyner.ws")
	if err != nil {
		log.Fatal(err)
	}
	defer aRecordFileHandle.Close()
	cfg.Origin = "iot.joyner.ws."
	err = dnsTmpl.Execute(aRecordFileHandle, cfg)
	if err != nil {
		log.Fatal("interpreting config: ", err)
	}

	PTRFileHandle, err := os.Create("db.64.0.10")
	if err != nil {
		log.Fatal(err)
	}
	defer PTRFileHandle.Close()
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
		// append to DNS files
		dnsName := fmt.Sprintf("%s.temphum.iot.joyner.ws.", dc.Name)
		entry := fmt.Sprintf("%s\tA\t%s\n", dnsName, dc.IpAddress)
		aRecordFileHandle.Write([]byte(entry))
		// TODO: actually parse the IP address with a library, maybe?  :)
		entry = fmt.Sprintf("%s\tPTR\t%s\n", strings.Split(dc.IpAddress, ".")[3], dnsName)
		PTRFileHandle.Write([]byte(entry))

		if strings.Contains(dc.MAC, "xx:xx") {
			continue
		}
		// append to leases file
		entry = fmt.Sprintf("%s,%s,%s,,,,\n", dc.MAC, dc.IpAddress, dc.Name)
		leasesFileHandle.Write([]byte(entry))

		// append to leases file
		entry = fmt.Sprintf("host %s { hardware ethernet %s; fixed-address %s; }\n", dc.Name, dc.MAC, dc.IpAddress)
		dhcpdconfFileHandle.Write([]byte(entry))

	}
}
