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
func createSerializedUDPFrame(opts UdpFrameOptions) ([]byte, error) {
	// EthernetHeader = 14
	// IP  Header = 20  <--+
	// UCP header = 8   <--+-- 1500 == MTU
	// Payload 6-1472   <--+
	// CRC 4

	//udpPayloadSize := len(opts.payloadBytes)
	log.Printf("TESTANDO ENVIO - payloadSize=%v", len(opts.payloadBytes))

	if len(opts.payloadBytes) > MaxUdpPayload { // MTU = 1500 = 1472 + 8 + 20
		log.Print("UDP payload bigger than 1472 bytes, will slipt in several packets")
		return udpPacketFragmentationControl(opts)
	} else {
		return udpOnePacket(opts)
	}

}

func udpPacketFragmentationControl(opts UdpFrameOptions) ([]byte, error) {
	// EthernetHeader = 14
	// IP  Header = 20  <--+
	// UDP header = 8   <--+-- 1500 == MTU
	// Payload 6-1472   <--+
	// CRC 4
	udpPayloadSize := len(opts.payloadBytes)

	numberOfFragments := udpPayloadSize / MaxUdpPayload
	lastUdpPayloadLength := (udpPayloadSize - MaxUdpPayload*numberOfFragments)

	if udpPayloadSize%1472 > 0 {
		numberOfFragments++
	}

	for i := 0; i < numberOfFragments; i++ {
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
		var flags layers.IPv4Flag
		if i < (numberOfFragments - 1) {
			flags = layers.IPv4MoreFragments
		} else {
			flags = layers.IPv4DontFragment
		}

		ip = &layers.IPv4{
			SrcIP:      opts.sourceIP,
			DstIP:      opts.destIP,
			Protocol:   layers.IPProtocolUDP,
			Version:    4,
			TTL:        32,
			FragOffset: IpFragOffsetStep * uint16(i),
			Flags:      flags,
		}
		var udp *layers.UDP
		if i == 0 {
			udp = &layers.UDP{
				SrcPort: layers.UDPPort(opts.sourcePort),
				DstPort: layers.UDPPort(opts.destPort),
				Length:  calculateUdpPayloadSize(lastUdpPayloadLength, i, numberOfFragments),
			}
		}

		udp.SetNetworkLayerForChecksum(ip)

		startSlicePos := i * IpFragOffsetStep * 8
		stopSlicePos := (i + 1) * IpFragOffsetStep * 8

		if stopSlicePos > len(opts.payloadBytes) {
			stopSlicePos = len(opts.payloadBytes)
		}

		payloadFragmented := opts.payloadBytes[startSlicePos:stopSlicePos]

		if i == 0 {
			err := gopacket.SerializeLayers(buf, serializeOpts, eth, ip, udp, gopacket.Payload(payloadFragmented))
			if err != nil {
				return nil, err
			}
		} else {
			err := gopacket.SerializeLayers(buf, serializeOpts, eth, ip, gopacket.Payload(payloadFragmented))
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func calculateUdpPayloadSize(lastUdpPayloadLength int, fragIdx int, numberOfFragments int) uint16 {
	if fragIdx < numberOfFragments-1 {
		return MaxUdpPayload
	} else {
		return uint16(lastUdpPayloadLength)
	}
}

func udpOnePacket(opts UdpFrameOptions) ([]byte, error) {

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
	return buf.Bytes(), nil
}
