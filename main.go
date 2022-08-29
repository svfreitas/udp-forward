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
	//---------------------
	handle, err := pcap.OpenLive(config.InterfaceName, int32(config.MaxPacketSize), false, pcap.BlockForever)

	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	addr := net.UDPAddr{
		Port: int(config.UdpPortReceiver),
		IP:   config.IpAddressReceiver,
	}
	log.Printf("[%v]", addr)
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
		log.Printf("[%v][%d]%v", remote, rlen, buf[:rlen])
		handlePacket2(handle, buf[:rlen], remote, config)
	}
	// Do stuff with the read bytes
	//---------------------
	// handle, err := pcap.OpenLive(config.InterfaceName, int32(config.MaxPacketSize), false, pcap.BlockForever)

	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer handle.Close()

	// bpfFilter := fmt.Sprintf("udp and port %d", config.UdpPortReceiver)

	// if err := handle.SetBPFFilter(bpfFilter); err != nil {
	// 	log.Fatal(err)
	// }

	// packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	// for packet := range packetSource.Packets() {
	// 	//start := time.Now()
	// 	handlePacket(handle, packet, config)
	// 	//elapsed := time.Since(start)
	// 	//log.Printf("trap redirection takes %s", elapsed)
	// }

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
		for i, frame := range sliceFrameBytes {
			log.Printf("frame[%d] = %v", i, frame)
			if err := handle.WritePacketData(frame); err != nil {
				log.Printf("Error Writing UDP data to destination %s : %s ", destination.IpAddress.String(), err)
			}
		}
	}
}

// func handlePacket(handle *pcap.Handle, packet gopacket.Packet, config *Config) {
// 	var udpFrameOptions UdpFrameOptions

// 	udpFrameOptions.sourceMac = config.MacAddressReceiver
// 	log.Print("-------------------------------------")
// 	log.Printf("udpFrameOptions.sourceMac = %v", udpFrameOptions.sourceMac)

// 	if netw := packet.NetworkLayer(); netw != nil {
// 		srcN, _ := netw.NetworkFlow().Endpoints()
// 		if transp := packet.TransportLayer(); transp != nil {
// 			if app := packet.ApplicationLayer(); app != nil {
// 				data := app.Payload()
// 				udpFrameOptions.sourceIP = net.ParseIP(srcN.String())
// 				udpFrameOptions.payloadBytes = data
// 				log.Printf("udpFrameOptions.sourceIP = %v", udpFrameOptions.sourceIP)
// 			}
// 		}
// 	}

// 	for _, destination := range config.Destinations {
// 		udpFrameOptions.destIP = destination.IpAddress
// 		udpFrameOptions.destMac = destination.MacAddress
// 		udpFrameOptions.destPort = destination.Port
// 		udpFrameOptions.isIPv6 = false

// 		log.Printf("udpFrameOptions.destIP = %v", udpFrameOptions.destIP)
// 		log.Printf("udpFrameOptions.destPort = %v", udpFrameOptions.destPort)
// 		log.Printf("udpFrameOptions.destMac = %v", udpFrameOptions.destMac)

// 		frameBytes, err := createSerializedUDPFrame(udpFrameOptions)

// 		if err != nil {
// 			log.Printf("Error serializing UDP frame to send to destination %s : %s", destination.IpAddress.String(), err)
// 			continue
// 		}

// 		if err := handle.WritePacketData(frameBytes); err != nil {
// 			log.Printf("Error Writing UDP data to destination %s : %s ", destination.IpAddress.String(), err)
// 		}
// 	}
// }
