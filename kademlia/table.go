package kademlia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kade/crypto"
	"kade/utils"
	"log"
	mrand "math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bucketSize      = 16 // Kademlia bucket size
	maxReplacements = 10 // Size of per-bucket replacement list

	hashBits          = len(crypto.Hash{}) * 8
	nBuckets          = hashBits / 15       // Number of buckets
	bucketMinDistance = hashBits - nBuckets // Log distance of closest bucket

	queryNodes         = 10
	refreshInterval    = 30 * time.Second
	revalidateInterval = 10 * time.Second
	copyNodesInterval  = 10 * time.Second
	seedMinTableTime   = 5 * time.Minute
	seedCount          = 30
	seedMaxAge         = 5 * 24 * time.Hour

	// MaxNeighbors is the maximum number of neighbor nodes in a Neighbors packet.
	MaxNeighbors = 12
)

// Table is the 'node table', a Kademlia-like index of neighbor nodes. The table keeps
// itself up-to-date by verifying the liveness of neighbors and requesting their node
// records when announcements of a new record version are received.
type Table struct {
	selfID    ID
	buckets   [nBuckets]*bucket // index of known nodes by distance
	bootnodes []*Node
	rand      *mrand.Rand // source of randomness, periodically reseeded

	udpConn *net.UDPConn

	closeReq chan struct{}
}

func NewTable(selfID ID, bootnodes []*Node, udpConn *net.UDPConn) (*Table, error) {
	tab := &Table{
		selfID:    selfID,
		bootnodes: bootnodes,
		rand:      mrand.New(mrand.NewSource(0)),
		udpConn:   udpConn,
		closeReq:  make(chan struct{}),
	}
	for i := range tab.buckets {
		tab.buckets[i] = &bucket{
			entries:      []*Node{},
			replacements: []*Node{},
		}
	}
	go tab.loop()
	go tab.HandleUdpPacket()
	return tab, nil
}

func FromNodeIDStr(bootnode string) (n *Node, err error) {
	a := strings.Split(strings.Split(bootnode, "//")[1], "@")
	addr := strings.Split(a[1], ":")
	ip := net.IP{}
	_ = ip.UnmarshalText([]byte(addr[0]))
	port, _ := strconv.Atoi(addr[1])
	id, err := ParseID(utils.Bytes2Hex(crypto.Keccak256(utils.FromHex(a[0]))))
	if err != nil {
		return
	}
	n = &Node{
		ID:      id,
		IP:      ip,
		Port:    port,
		addedAt: time.Now(),
	}
	return
}

// Ping some nodes
func (tab *Table) doRevalidate() {
	ping := Ping{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(ping.Kind())
	buf.Write(ping.Encode())
	nodesCountInBuckets := 0

	for _, b := range &tab.buckets {
		if len(b.entries) == 0 {
			continue
		}
		// 总是ping第一个节点
		node := b.entries[0]

		_, _ = tab.udpConn.WriteToUDP(buf.Bytes(), &net.UDPAddr{IP: node.IP, Port: node.Port})
		nodesCountInBuckets++
		b.entries[0].liveChecks++
		if b.entries[0].liveChecks > 3 {
			b.mutex.Lock()
			b.entries = deleteNode(b.entries, node)
			log.Printf("Removed dead node, id: %x, ip: %v:%v", node.ID, node.IP, node.Port)
			b.mutex.Unlock()
		}
	}
	if nodesCountInBuckets == 0 {
		for _, b := range tab.bootnodes {
			_, _ = tab.udpConn.WriteToUDP(buf.Bytes(), &net.UDPAddr{IP: b.IP, Port: b.Port})
		}
	}
}

func (tab *Table) findNode() {
	nodes := tab.NodeQueryList()
	ping := Findnode{ReqID: tab.selfID}
	b := new(bytes.Buffer)
	b.WriteByte(ping.Kind())
	b.Write(ping.Encode())
	for _, n := range nodes {
		_, _ = tab.udpConn.WriteToUDP(b.Bytes(), &net.UDPAddr{IP: n.IP, Port: n.Port})
	}
}

func (tab *Table) storageNode() {
	// save node to somewhere
}

func (tab *Table) loop() {
	var (
		revalidate = time.NewTicker(revalidateInterval)
		refresh    = time.NewTicker(refreshInterval)
		copyNodes  = time.NewTicker(copyNodesInterval)
		logTicker  = time.NewTicker(revalidateInterval)
	)

	// Start initial.
	tab.doRevalidate()
	tab.findNode()
	tab.storageNode()

loop:
	for {
		select {
		case <-revalidate.C:
			tab.doRevalidate()
		case <-refresh.C:
			tab.findNode()
		case <-copyNodes.C:
			tab.storageNode()
		case <-logTicker.C:
			nodesCount := 0
			for _, b := range &tab.buckets {
				nodesCount += len(b.entries)
			}
			fmt.Println("nodes count:", nodesCount)
		case <-tab.closeReq:
			break loop
		}
	}
}

func (tab *Table) NodeQueryList() (nodes []*Node) {
	for _, b := range &tab.buckets {
		for _, node := range b.entries {
			if len(nodes) == queryNodes {
				return
			}
			nodes = append(nodes, node)
		}
	}

	return
}

func (tab *Table) HandleUdpPacket() {
	var data [maxPacketSize]byte
	for {
		nbytes, addr, err := tab.udpConn.ReadFromUDP(data[:]) // 接收数据
		if err != nil {
			log.Println("read from UDP failed, err:", err)
			continue
		}
		var req Packet
		switch ptype := data[0]; ptype {
		case PingPacket:
			req = new(Ping)
		case PongPacket:
			req = new(Pong)
		case FindnodePacket:
			req = new(Findnode)
		case NeighborsPacket:
			req = new(Neighbors)
		case ENRRequestPacket:
			req = new(ENRRequest)
		case ENRResponsePacket:
			req = new(ENRResponse)
		}
		err = json.Unmarshal(data[1:nbytes], req)
		if err != nil {
			continue
		}
		go tab.HandlePacket(req, addr)
		log.Println("read from UDP:", req.Name(), addr)
	}
}

func (tab *Table) HandlePacket(req Packet, addr *net.UDPAddr) {
	switch req.Kind() {
	case PingPacket:
		_ = tab.sendPong(addr)
		packet := req.(*Ping)
		n := &Node{
			ID:   packet.ReqID,
			IP:   addr.IP,
			Port: addr.Port,
		}
		tab.addSeenNode(n, true)
	case PongPacket:
		packet := req.(*Pong)
		n := &Node{
			ID:   packet.ReqID,
			IP:   addr.IP,
			Port: addr.Port,
		}
		tab.addSeenNode(n, true)
	case FindnodePacket:
		packet := req.(*Findnode)
		nodes := tab.findClosestNodes(packet.ReqID, MaxNeighbors)
		_ = tab.sendNeighbors(nodes, addr)

	case NeighborsPacket:
		packet := req.(*Neighbors)
		for _, n := range packet.NeighborNodes {
			tab.addSeenNode(n, false)
		}

	case ENRRequestPacket:
		//req = new(ENRRequest)
	case ENRResponsePacket:
		//req = new(ENRResponse)
	}
}

func (tab *Table) findClosestNodes(sourceID ID, count int) (nodes []*Node) {
	distance := make(map[int][]*Node)
	for _, b := range &tab.buckets {
		for _, node := range b.entries {
			if node.liveChecks > 0 {
				continue
			}
			d := LogDist(sourceID, node.ID)
			if distance[d] == nil {
				distance[d] = []*Node{}
			}
			distance[d] = append(distance[d], node)
		}
	}

	var keys []int
	for k := range distance {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	nodes = []*Node{}
	for _, d := range keys {
		nodes = append(nodes, distance[d]...)
		if len(nodes) >= count {
			break
		}
	}

	r := len(nodes)
	if r > count {
		r = count
	}
	return nodes[:r]
}

func (tab *Table) addSeenNode(n *Node, resetLiveChecks bool) {
	if n.ID == tab.selfID {
		return
	}

	b := tab.bucket(n.ID)
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if i := contains(b.entries, n.ID); i != -1 {
		// Already in bucket, move to the end.
		if resetLiveChecks {
			b.entries[i].liveChecks = 0
		}
		n.liveChecks = b.entries[i].liveChecks
		b.entries = append(append(b.entries[:i], b.entries[i+1:]...), n)
		return
	}

	if len(b.entries) >= bucketSize {
		// Bucket full, maybe add as replacement.
		tab.addReplacement(b, n)
		return
	}

	// Add to end of bucket:
	b.entries = append(b.entries, n)
	b.replacements = deleteNode(b.replacements, n)
	n.addedAt = time.Now()
	log.Printf("Add new node to bucket, addr: %v:%v", n.IP, n.Port)
}

func (tab *Table) addReplacement(b *bucket, n *Node) {
	for _, e := range b.replacements {
		if e.ID == n.ID {
			return // already in list
		}
	}

	b.replacements, _ = pushNode(b.replacements, n, maxReplacements)
}

func (tab *Table) sendUDPPacket(packet []byte, addr *net.UDPAddr) (err error) {
	_, err = tab.udpConn.WriteToUDP(packet, &net.UDPAddr{IP: addr.IP, Port: addr.Port})
	return
}

func (tab *Table) sendPing(addr *net.UDPAddr) (err error) {
	ping := Ping{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(ping.Kind())
	buf.Write(ping.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

func (tab *Table) sendPong(addr *net.UDPAddr) (err error) {
	pong := Pong{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(pong.Kind())
	buf.Write(pong.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

func (tab *Table) sendFindnode(addr *net.UDPAddr) (err error) {
	findnode := Findnode{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(findnode.Kind())
	buf.Write(findnode.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

func (tab *Table) sendNeighbors(nodes []*Node, addr *net.UDPAddr) (err error) {
	neighbors := Neighbors{NeighborNodes: nodes}
	buf := new(bytes.Buffer)
	buf.WriteByte(neighbors.Kind())
	buf.Write(neighbors.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

func (tab *Table) sendENRRequest(addr *net.UDPAddr) (err error) {
	enrRequest := ENRRequest{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(enrRequest.Kind())
	buf.Write(enrRequest.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

func (tab *Table) sendENRResponse(addr *net.UDPAddr) (err error) {
	enrResponse := ENRResponse{ReqID: tab.selfID}
	buf := new(bytes.Buffer)
	buf.WriteByte(enrResponse.Kind())
	buf.Write(enrResponse.Encode())
	err = tab.sendUDPPacket(buf.Bytes(), addr)
	return
}

// bucket contains nodes, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct {
	entries      []*Node // live entries, sorted by time of last contact
	replacements []*Node // recently seen nodes to be used if revalidation fails

	mutex sync.Mutex
}

func contains(ns []*Node, id ID) int {
	for i, n := range ns {
		if n.ID == id {
			return i
		}
	}
	return -1
}

// deleteNode removes n from list.
func deleteNode(list []*Node, n *Node) []*Node {
	for i := range list {
		if list[i].ID == n.ID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// pushNode adds n to the front of list, keeping at most max items.
func pushNode(list []*Node, n *Node, max int) ([]*Node, *Node) {
	if len(list) < max {
		list = append(list, nil)
	}
	removed := list[len(list)-1]
	copy(list[1:], list)
	list[0] = n
	return list, removed
}
