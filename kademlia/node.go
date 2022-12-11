package kademlia

import (
	"encoding/hex"
	"fmt"
	"github.com/labstack/gommon/log"
	"kade/params"
	"net"
	"strings"
	"time"
)

// Node represents a host on the network.
// The fields of Node may not be modified.
type Node struct {
	ID             ID
	IP             net.IP
	Port           int
	addedAt        time.Time // time when the Node was added to the table
	livenessChecks uint      // how often liveness was checked
}

// ID is a unique identifier for each node.
type ID [32]byte

func ParseID(in string) (id ID, err error) {
	b, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %d hex chars", len(id)*2)
	}
	copy(id[:], b)
	return id, nil
}

func GetBootnodes(ignoreID ID) (bootnodes []*Node) {
	for _, n := range params.MainnetBootnodes {
		node, err := FromNodeIDStr(n)
		if err != nil {
			log.Error(err)
			continue
		}
		if node.ID == ignoreID {
			continue
		}
		bootnodes = append(bootnodes, node)
	}
	return
}
