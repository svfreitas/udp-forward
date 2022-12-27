package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/j-keck/arping"
)

type rawConfig struct {
	InterfaceName      string   `json:"interface_name"`
	LogFileLocation    string   `json:"log_file_location"`
	LogFileSize        int      `json:"log_file_size"`
	LogFileMaxBackups  int      `json:"log_file_max_backups"`
	LogLevelConfigPort int      `json:"log_level_config_port"`
	IpAddressReceiver  string   `json:"ip_address_receiver"`
	DefaultGateway     string   `json:"default_gateway"`
	MaxPacketSize      int      `json:"max_packet_size"`
	Destinations       []string `json:"destinations"`
}

type Config struct {
	InterfaceName           string
	LogLevelConfigPort      int
	LogFileLocation         string
	LogFileSize             int
	LogFileMaxBackups       int
	IpAddressReceiver       net.IP
	UdpPortReceiver         uint16
	MacAddresDefaultGateway net.HardwareAddr
	MacAddressReceiver      net.HardwareAddr
	MaxPacketSize           int
	Destinations            []Destination
}

type Destination struct {
	IpAddress  net.IP
	MacAddress net.HardwareAddr
	Port       uint16
}

func LoadConfiguration(filename string) (*Config, error) {

	var config Config
	var rawConfig rawConfig
	var err error

	content, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(content), &rawConfig)

	if err != nil {
		log.Fatalf("Configuration file problem : %s", err)
	}

	config.LogLevelConfigPort = rawConfig.LogLevelConfigPort
	if config.LogLevelConfigPort <= 0 {
		log.Fatal("log_level_config_port bad format")
	}

	config.LogFileLocation = rawConfig.LogFileLocation
	if config.LogFileLocation == "" {
		config.LogFileLocation = "./"
	} else if strings.HasSuffix(config.LogFileLocation, "/") == false {
		config.LogFileLocation += "/"
	}

	config.LogFileMaxBackups = rawConfig.LogFileMaxBackups
	if config.LogFileMaxBackups == 0 {
		config.LogFileMaxBackups = 3
	}

	config.LogFileSize = rawConfig.LogFileSize
	if config.LogFileSize == 0 {
		config.LogFileSize = 10
	}

	config.InterfaceName = rawConfig.InterfaceName
	if config.InterfaceName == "" {
		log.Fatal("interface_name missing")
	}

	host, port, err := net.SplitHostPort(rawConfig.IpAddressReceiver)
	if err != nil {
		log.Fatal("ip_address_receiver bad format")
	}

	config.IpAddressReceiver = net.ParseIP(host)
	if config.IpAddressReceiver == nil {
		log.Fatal("ip_address_receiver bad format for host")
	}

	// find MAC address of receiver IP
	netInterface, err := net.InterfaceByName(config.InterfaceName)
	if err != nil {
		log.Fatalf("MAC address resolution problem for Receiver interface_name: %s", err)
	} else {
		config.MacAddressReceiver = netInterface.HardwareAddr
	}

	portUint, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		log.Fatal("Port bad format")
	}
	config.UdpPortReceiver = uint16(portUint)

	config.MaxPacketSize = rawConfig.MaxPacketSize

	dg := net.ParseIP(rawConfig.DefaultGateway)
	if dg == nil {
		log.Fatal("DefaultGateway bad format")
	}

	if hwAddr, _, err := arping.Ping(dg); err != nil {
		log.Fatalf("MAC address resolution problem for DefaultGateway: %s", err)
	} else {
		config.MacAddresDefaultGateway = hwAddr
	}

	for _, rawDest := range rawConfig.Destinations {
		var dest Destination
		host, port, err := net.SplitHostPort(rawDest)
		if err != nil {
			log.Fatalf("Host Port bad format for Destination: %s", rawDest)
		}

		dest.IpAddress = net.ParseIP(host)
		if dest.IpAddress == nil {
			log.Fatalf("IpAddressDestination bad format for Destination:%s", rawDest)
		}

		uPort, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			log.Fatalf("Port bad format for Destination: %s", rawDest)
		}
		dest.Port = uint16(uPort)

		dest.MacAddress = findMacAddress(dest.IpAddress, host, netInterface, config.MacAddresDefaultGateway)

		config.Destinations = append(config.Destinations, dest)
	}

	return &config, nil

}

func findMacAddress(ipAddress net.IP, host string, netInterface *net.Interface, defaulGatewayMacAddress net.HardwareAddr) net.HardwareAddr {
	var hwAddr net.HardwareAddr
	var err error
	var found bool

	if hwAddr, _, err = arping.Ping(ipAddress); err != nil {
		sliceAddresses, err := netInterface.Addrs()
		if err != nil {
			log.Printf("Failed to obtain addresses assigned to = %v with error = %v", netInterface, err)
		} else {
			log.Printf("sliceAddresses = %v", sliceAddresses)
			for _, v := range sliceAddresses {
				if strings.Contains(v.String(), host) {
					hwAddr = netInterface.HardwareAddr
					found = true
					log.Print("Host match as a local IP, setting hardware address as the interface HW adress")
					break
				}
			}
		}
		if !found {
			log.Printf("MAC address resolution problem for Destination: %s, setting Default Gateway MAC Address", ipAddress.String())
			hwAddr = defaulGatewayMacAddress
		}
	}
	return hwAddr
}
