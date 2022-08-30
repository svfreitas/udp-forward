package main

import (
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const MTU = 1500
const MaxUdpPayload = MTU - 28
const IpFragOffsetStep = (MTU - 20) / 8
const packetID uint16 = 12345

type UdpFrameOptions struct {
	sourceIP, destIP     net.IP
	sourcePort, destPort uint16
	sourceMac, destMac   net.HardwareAddr
	isIPv6               bool
	payloadBytes         []byte
}

type serializableNetworkLayer interface {
	gopacket.NetworkLayer
	SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error
}

// createSerializedUDPFrame creates an Ethernet frame encapsulating our UDP
// packet for injection to the local network
func createSerializedUDPFrame(opts UdpFrameOptions) ([][]byte, error) {
	// EthernetHeader = 14
	// IP  Header = 20  <--+
	// UCP header = 8   <--+-- 1500 == MTU
	// Payload 6-1472   <--+
	// CRC 4

	//udpPayloadSize := len(opts.payloadBytes)
	log.Printf("payloadSize=%v", len(opts.payloadBytes))

	if len(opts.payloadBytes) > MaxUdpPayload { // MTU = 1500 = 1472 + 8 + 20
		log.Print("UDP payload bigger than 1472 bytes, will be fragmented in several packets")
		return udpPacketFragmentationControl(opts)
	} else {
		return udpOnePacket(opts)
	}

}

func udpPacketFragmentationControl(opts UdpFrameOptions) ([][]byte, error) {
	// EthernetHeader = 14
	// IP  Header = 20  <--+
	// UDP header = 8   <--+-- 1500 == MTU
	// Payload 6-1472   <--+
	// CRC 4
	udpPayloadSize := len(opts.payloadBytes)

	factor := 1 + (udpPayloadSize - (MTU - 28))
	numberOfFragments := 1 + factor/(MTU-20)

	if factor%(MTU-20) > 0 {
		numberOfFragments++
	}

	slicePackets := make([][]byte, numberOfFragments)

	log.Printf("numberOfFragments=%v", numberOfFragments)

	//Used to calculate checksum and IP packet length
	udp2 := &layers.UDP{
		SrcPort: layers.UDPPort(opts.sourcePort),
		DstPort: layers.UDPPort(opts.destPort),
		Length:  uint16(len(opts.payloadBytes) + 8),
	}

	for i := 0; i < numberOfFragments; i++ {
		buf := gopacket.NewSerializeBuffer()
		log.Printf("packet %d", i)
		serializeOpts := gopacket.SerializeOptions{
			FixLengths:       false,
			ComputeChecksums: false,
		}
		serializeOpts2 := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		ethernetType := layers.EthernetTypeIPv4
		if opts.isIPv6 {
			ethernetType = layers.EthernetTypeIPv6
		}
		eth := &layers.Ethernet{
			SrcMAC:       opts.sourceMac,
			DstMAC:       opts.destMac,
			EthernetType: ethernetType,
		}

		var flags layers.IPv4Flag

		if i < numberOfFragments-1 {
			flags = layers.IPv4MoreFragments
			//log.Printf("IPv4MoreFragments")
		} else {
			flags = 0
			//log.Printf("flags = 0")
		}

		ip := &layers.IPv4{
			Id:         uint16(packetID),
			SrcIP:      opts.sourceIP,
			DstIP:      opts.destIP,
			Protocol:   layers.IPProtocolUDP,
			Version:    4,
			TTL:        64,
			FragOffset: IpFragOffsetStep * uint16(i),
			Flags:      flags,
			IHL:        5,
		}

		var udp *layers.UDP

		if i == 0 {
			//PseudoHeader used to calculate UDP Checksum
			ipPseudoHeader := &layers.IPv4{
				Id:         uint16(packetID),
				SrcIP:      opts.sourceIP,
				DstIP:      opts.destIP,
				Protocol:   layers.IPProtocolUDP,
				Version:    4,
				TTL:        64,
				FragOffset: 0,
				Flags:      0,
				IHL:        5,
				Length:     1500,
			}

			udp2.SetNetworkLayerForChecksum(ipPseudoHeader)
			err := gopacket.SerializeLayers(buf, serializeOpts2, eth, ipPseudoHeader, udp2, gopacket.Payload(opts.payloadBytes))

			//log.Printf("udp2.Checksum = %v", udp2.Checksum)

			if err != nil {
				return nil, err
			}

			udp = &layers.UDP{
				SrcPort:  layers.UDPPort(opts.sourcePort),
				DstPort:  layers.UDPPort(opts.destPort),
				Length:   udp2.Length,
				Checksum: udp2.Checksum,
			}
		}
		stopSlicePos := 0
		startSlicePos := 0

		if i == 0 {
			startSlicePos = 0
			stopSlicePos = (i+1)*IpFragOffsetStep*8 - 8 //1480 -8 = 1472
		} else {
			startSlicePos = i*IpFragOffsetStep*8 - 8
			stopSlicePos = (i+1)*IpFragOffsetStep*8 - 8
		}

		if stopSlicePos > len(opts.payloadBytes) {
			stopSlicePos = len(opts.payloadBytes)
		}
		//log.Printf("startSlicePos = %d, stopSlicePos = %d", startSlicePos, stopSlicePos)
		//log.Printf("len(payloadBytes) = %d", len(opts.payloadBytes))
		payloadFragmented := opts.payloadBytes[startSlicePos:stopSlicePos]
		log.Printf("len(payloadFragmented) = %d", len(payloadFragmented))
		//log.Printf("payloadFragmented = [%v]", payloadFragmented)

		if i == 0 {
			ip.Length = uint16(len(payloadFragmented) + 8 + 20)
		} else {
			ip.Length = uint16(len(payloadFragmented) + 20)
		}

		//log.Printf("ip = %v", ip)

		if i == 0 {

			udp.SetNetworkLayerForChecksum(ip)

			//Para calcular o IP checksum
			err := gopacket.SerializeLayers(buf, serializeOpts2, eth, ip, udp, gopacket.Payload(payloadFragmented))
			if err != nil {
				return nil, err
			}
			ip.Length = MTU
			udp.Length = udp2.Length
			udp.Checksum = udp2.Checksum
			//	log.Printf("**********\nip.checksum = %v\nip.Length = %v\nudp.Checksum=%v\nudp.Length=%v\n**********", ip.Checksum, ip.Length, udp.Checksum, udp.Length)

			err = gopacket.SerializeLayers(buf, serializeOpts, eth, ip, udp, gopacket.Payload(payloadFragmented))
			if err != nil {
				return nil, err
			}
		} else {
			err := gopacket.SerializeLayers(buf, serializeOpts2, eth, ip, gopacket.Payload(payloadFragmented))
			if err != nil {
				return nil, err
			}
		}
		slicePackets[i] = buf.Bytes()
		log.Printf("len(slicePackets) = [%d]", len(slicePackets))
	}

	return slicePackets, nil
}

func udpOnePacket(opts UdpFrameOptions) ([][]byte, error) {

	buf := gopacket.NewSerializeBuffer()
	serializeOpts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	ethernetType := layers.EthernetTypeIPv4
	if opts.isIPv6 {
		ethernetType = layers.EthernetTypeIPv6
	}
	eth := &layers.Ethernet{
		SrcMAC:       opts.sourceMac,
		DstMAC:       opts.destMac,
		EthernetType: ethernetType,
	}
	var ip serializableNetworkLayer

	if !opts.isIPv6 {
		ip = &layers.IPv4{
			SrcIP:    opts.sourceIP,
			DstIP:    opts.destIP,
			Protocol: layers.IPProtocolUDP,
			Version:  4,
			TTL:      32,
		}
	} else {
		ip = &layers.IPv6{
			SrcIP:      opts.sourceIP,
			DstIP:      opts.destIP,
			NextHeader: layers.IPProtocolUDP,
			Version:    6,
			HopLimit:   32,
		}
		ip.LayerType()
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(opts.sourcePort),
		DstPort: layers.UDPPort(opts.destPort),
	}
	udp.SetNetworkLayerForChecksum(ip)
	err := gopacket.SerializeLayers(buf, serializeOpts, eth, ip, udp, gopacket.Payload(opts.payloadBytes))
	if err != nil {
		return nil, err
	}
	slicePackets := make([][]byte, 1)
	slicePackets[0] = buf.Bytes()
	return slicePackets, nil
}
