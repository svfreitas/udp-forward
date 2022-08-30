package main

import (
	"flag"
	"log"
	"net"

	"github.com/google/gopacket/pcap"
)

func main() {

	filename := flag.String("f", "", "configuration file")
	flag.Parse()
	if *filename == "" {
		flag.Usage()
		log.Fatal("Must provide configuration file: udp-forwarder -f <filename>")
	}
	config, err := LoadConfiguration(*filename)

	if err != nil {
		log.Printf("Unable to load configuration, error :%s", err)
		return
	}

	handle, err := pcap.OpenLive(config.InterfaceName, int32(config.MaxPacketSize), false, pcap.BlockForever)

	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	addr := net.UDPAddr{
		Port: int(config.UdpPortReceiver),
		IP:   config.IpAddressReceiver,
	}
	log.Printf("Listening on [%v]", addr)
	conn, err := net.ListenUDP("udp", &addr) // code does not block here
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var buf = make([]byte, 65527)
	for {
		rlen, remote, err := conn.ReadFromUDP(buf)

		if err != nil {
			panic(err)
		}
		log.Printf("Data read from [%v], length %d", remote, rlen)
		handlePacket2(handle, buf[:rlen], remote, config)
	}
}

func handlePacket2(handle *pcap.Handle, payload []byte, remote *net.UDPAddr, config *Config) {
	var udpFrameOptions UdpFrameOptions

	udpFrameOptions.sourceMac = config.MacAddressReceiver
	log.Print("-------------------------------------")
	log.Printf("udpFrameOptions.sourceMac = %v", udpFrameOptions.sourceMac)

	udpFrameOptions.sourceIP = remote.IP
	udpFrameOptions.sourcePort = uint16(remote.Port)
	udpFrameOptions.payloadBytes = payload
	log.Printf("udpFrameOptions.sourceIP = %v", udpFrameOptions.sourceIP)

	for _, destination := range config.Destinations {
		udpFrameOptions.destIP = destination.IpAddress
		udpFrameOptions.destMac = destination.MacAddress
		udpFrameOptions.destPort = destination.Port
		udpFrameOptions.isIPv6 = false

		log.Printf("udpFrameOptions.destIP = %v", udpFrameOptions.destIP)
		log.Printf("udpFrameOptions.destPort = %v", udpFrameOptions.destPort)
		log.Printf("udpFrameOptions.destMac = %v", udpFrameOptions.destMac)

		sliceFrameBytes, err := createSerializedUDPFrame(udpFrameOptions)

		if err != nil {
			log.Printf("Error serializing UDP frame to send to destination %s : %s", destination.IpAddress.String(), err)
			continue
		}
		log.Printf("len(sliceFrameBytes) = %d", len(sliceFrameBytes))
		for _, frame := range sliceFrameBytes {
			//log.Printf("Sending frame[%d] = %v", i, frame)
			if err := handle.WritePacketData(frame); err != nil {
				log.Printf("Error Writing UDP data to destination %s : %s ", destination.IpAddress.String(), err)
			}
		}
	}
}
