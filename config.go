package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"strconv"

	"github.com/j-keck/arping"
)

type rawConfig struct {
	InterfaceName     string `json:"interface_name"`
	IpAddressReceiver string `json:"ip_address_receiver"`
	//	MacAddressReceiver string   `json:"mac_address_receiver"`
	DefaultGateway string   `json:"default_gateway"`
	MaxPacketSize  int      `json:"max_packet_size"`
	Destinations   []string `json:"destinations"`
}

type Config struct {
	InterfaceName           string
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

	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(content), &rawConfig)

	if err != nil {
		log.Fatalf("Configuration file problem : %s", err)
	}

	config.InterfaceName = rawConfig.InterfaceName

	host, port, err := net.SplitHostPort(rawConfig.IpAddressReceiver)
	if err != nil {
		log.Fatal("Host Port bad format")
	}

	config.IpAddressReceiver = net.ParseIP(host)
	if config.IpAddressReceiver == nil {
		log.Fatal("IpAddressReceiver bad format")
	}

	// find MAC address of receiver IP
	netInterface, err := net.InterfaceByName(config.InterfaceName)
	if err != nil {
		log.Printf("MAC address resolution problem for Receiver: %s", err)
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

		var hwAddr net.HardwareAddr
		if hwAddr, _, err = arping.Ping(dest.IpAddress); err != nil {
			log.Printf("MAC address resolution problem for Destination: %s, setting Default Gateway MAC Address", dest.IpAddress.String())
			hwAddr = config.MacAddresDefaultGateway
		}
		dest.MacAddress = hwAddr

		config.Destinations = append(config.Destinations, dest)
	}

	return &config, nil

}
