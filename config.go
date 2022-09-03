package main

import (
	"encoding/json"
	"net"
	"os"
	"strconv"

	"github.com/j-keck/arping"
)

type rawConfig struct {
	InterfaceName      string   `json:"interface_name"`
	LogLevelConfigPort int      `json:"log_level_config_port"`
	IpAddressReceiver  string   `json:"ip_address_receiver"`
	DefaultGateway     string   `json:"default_gateway"`
	MaxPacketSize      int      `json:"max_packet_size"`
	Destinations       []string `json:"destinations"`
}

type Config struct {
	InterfaceName           string
	LogLevelConfigPort      int
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
		slogger.Fatalf("Configuration file problem : %s", err)
	}

	config.LogLevelConfigPort = rawConfig.LogLevelConfigPort
	if config.LogLevelConfigPort <= 0 {
		slogger.Fatal("log_level_config_port bad format")
	}

	config.InterfaceName = rawConfig.InterfaceName
	if config.InterfaceName == "" {
		slogger.Fatal("interface_name missing")
	}

	host, port, err := net.SplitHostPort(rawConfig.IpAddressReceiver)
	if err != nil {
		slogger.Fatal("ip_address_receiver bad format")
	}

	config.IpAddressReceiver = net.ParseIP(host)
	if config.IpAddressReceiver == nil {
		slogger.Fatal("ip_address_receiver bad format for host")
	}

	// find MAC address of receiver IP
	netInterface, err := net.InterfaceByName(config.InterfaceName)
	if err != nil {
		slogger.Fatalf("MAC address resolution problem for Receiver interface_name: %s", err)
	} else {
		config.MacAddressReceiver = netInterface.HardwareAddr
	}

	portUint, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		slogger.Fatal("Port bad format")
	}
	config.UdpPortReceiver = uint16(portUint)

	config.MaxPacketSize = rawConfig.MaxPacketSize

	dg := net.ParseIP(rawConfig.DefaultGateway)
	if dg == nil {
		slogger.Fatal("DefaultGateway bad format")
	}

	if hwAddr, _, err := arping.Ping(dg); err != nil {
		slogger.Fatalf("MAC address resolution problem for DefaultGateway: %s", err)
	} else {
		config.MacAddresDefaultGateway = hwAddr
	}

	for _, rawDest := range rawConfig.Destinations {
		var dest Destination
		host, port, err := net.SplitHostPort(rawDest)
		if err != nil {
			slogger.Fatalf("Host Port bad format for Destination: %s", rawDest)
		}

		dest.IpAddress = net.ParseIP(host)
		if dest.IpAddress == nil {
			slogger.Fatalf("IpAddressDestination bad format for Destination:%s", rawDest)
		}

		uPort, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			slogger.Fatalf("Port bad format for Destination: %s", rawDest)
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
			slogger.Warnf("Failed to obtain addresses assigned to = %v with error = %v", netInterface, err)
		} else {
			slogger.Debugf("sliceAddresses = %v", sliceAddresses)
			for _, v := range sliceAddresses {
				hostFromList, _, err := net.SplitHostPort(v.String())
				if err != nil {
					slogger.Warnf("Failed to parse the addresses assigned to = %v with error = %v", netInterface, err)
				} else {
					slogger.Debugf("hostFromList = %v", hostFromList)
					if hostFromList == host {
						hwAddr = netInterface.HardwareAddr
						found = true
						slogger.Debug("Host match as a local IP, setting hardware address as the interface HW adress")
						break
					}
				}
			}
		}
		if !found {
			slogger.Warnf("MAC address resolution problem for Destination: %s, setting Default Gateway MAC Address", ipAddress.String())
			hwAddr = defaulGatewayMacAddress
		}
	}
	return hwAddr
}
