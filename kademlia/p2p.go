package kademlia

import (
	"log"
	"net"
)

const (
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't match.
	maxPacketSize = 2560
)

type ReadPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

func ListenUDP(port int) (udpConn *net.UDPConn) {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: port,
	})
	if err != nil {
		log.Fatal("listen UDP failed, err:", err)
	}
	return
}
