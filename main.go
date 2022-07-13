package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/google/gopacket"
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

	bpfFilter := fmt.Sprintf("udp and port %d", config.UdpPortReceiver)
	if err := handle.SetBPFFilter(bpfFilter); err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for packet := range packetSource.Packets() {
		//start := time.Now()
		handlePacket(handle, packet, config)
		//elapsed := time.Since(start)
		//log.Printf("trap redirection takes %s", elapsed)
	}

}

func handlePacket(handle *pcap.Handle, packet gopacket.Packet, config *Config) {
	var udpFrameOptions UdpFrameOptions

	udpFrameOptions.sourceMac = config.MacAddressReceiver
	log.Print("-------------------------------------")
	log.Printf("udpFrameOptions.sourceMac = %v", udpFrameOptions.sourceMac)

	if netw := packet.NetworkLayer(); netw != nil {
		srcN, _ := netw.NetworkFlow().Endpoints()
		if transp := packet.TransportLayer(); transp != nil {
			if app := packet.ApplicationLayer(); app != nil {
				data := app.Payload()
				udpFrameOptions.sourceIP = net.ParseIP(srcN.String())
				udpFrameOptions.payloadBytes = data
				log.Printf("udpFrameOptions.sourceIP = %v", udpFrameOptions.sourceIP)
			}
		}
	}

	for _, destination := range config.Destinations {
		udpFrameOptions.destIP = destination.IpAddress
		udpFrameOptions.destMac = destination.MacAddress
		udpFrameOptions.destPort = destination.Port
		udpFrameOptions.isIPv6 = false

		log.Printf("udpFrameOptions.destIP = %v", udpFrameOptions.destIP)
		log.Printf("udpFrameOptions.destPort = %v", udpFrameOptions.destPort)
		log.Printf("udpFrameOptions.destMac = %v", udpFrameOptions.destMac)

		frameBytes, err := createSerializedUDPFrame(udpFrameOptions)

		if err != nil {
			log.Printf("Error serializing UDP frame to send to destination %s : %s", destination.IpAddress.String(), err)
		}

		if err := handle.WritePacketData(frameBytes); err != nil {
			log.Printf("Error Writing UDP data to destination %s : %s ", destination.IpAddress.String(), err)
		}
	}
}
