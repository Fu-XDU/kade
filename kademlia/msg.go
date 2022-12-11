package kademlia

import "encoding/json"

// RPC packet types
const (
	PingPacket = iota + 1 // zero is 'reserved'
	PongPacket
	FindnodePacket
	NeighborsPacket
	ENRRequestPacket
	ENRResponsePacket
)

type Packet interface {
	Name() string
	Kind() byte
	Encode() []byte
}

// RPC request structures
type (
	Ping struct {
		ReqID ID
	}

	// Pong is the reply to ping.
	Pong struct {
		ReqID ID
	}

	// Findnode is a query for nodes close to the given target.
	Findnode struct {
		ReqID ID
	}

	// Neighbors is the reply to findnode.
	Neighbors struct {
		//ReqID         ID
		NeighborNodes []*Node
	}

	// ENRRequest queries for the remote node's record.
	ENRRequest struct {
		ReqID ID
	}

	// ENRResponse is the reply to ENRRequest.
	ENRResponse struct {
		ReqID ID
	}
)

func (req *Ping) Name() string { return "PING/v4" }
func (req *Ping) Kind() byte   { return PingPacket }
func (req *Ping) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}

func (req *Pong) Name() string { return "PONG/v4" }
func (req *Pong) Kind() byte   { return PongPacket }
func (req *Pong) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}

func (req *Findnode) Name() string { return "FINDNODE/v4" }
func (req *Findnode) Kind() byte   { return FindnodePacket }
func (req *Findnode) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}

func (req *Neighbors) Name() string { return "NEIGHBORS/v4" }
func (req *Neighbors) Kind() byte   { return NeighborsPacket }
func (req *Neighbors) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}

func (req *ENRRequest) Name() string { return "ENRREQUEST/v4" }
func (req *ENRRequest) Kind() byte   { return ENRRequestPacket }
func (req *ENRRequest) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}

func (req *ENRResponse) Name() string { return "ENRRESPONSE/v4" }
func (req *ENRResponse) Kind() byte   { return ENRResponsePacket }
func (req *ENRResponse) Encode() []byte {
	m, _ := json.Marshal(req)
	return m
}
