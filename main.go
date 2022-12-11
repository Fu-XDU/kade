package main

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli"
	"kade/crypto"
	"kade/kademlia"
	"kade/utils"
	"os"
)

const (
	clientIdentifier = "P2P client"
	clientVersion    = "1.0.0"
	clientUsage      = "P2P client"
)

var (
	app       = cli.NewApp()
	baseFlags = []cli.Flag{
		utils.P2PPortFlag,
	}
)

func init() {
	app.Action = startClient
	app.Name = clientIdentifier
	app.Version = clientVersion
	app.Usage = clientUsage
	app.Commands = []cli.Command{}
	app.Flags = append(app.Flags, baseFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func startClient(ctx *cli.Context) error {
	_, pk := crypto.LoadKey()

	pubKey := crypto.GetNodeIDFromPubKey(pk)
	// 将NodeID做SHA-256得到target
	// unique identifier for each node.
	nodeID, err := kademlia.ParseID(utils.Bytes2Hex(crypto.Keccak256(pubKey)))
	if err != nil {
		log.Fatal(err)
	}

	conn := kademlia.ListenUDP(ctx.Int("p"))
	log.Print("Started P2P networking", " nodeID=enode://", utils.Bytes2Hex(pubKey)+"@127.0.0.1:", ctx.Int("p"))
	bootnodes := kademlia.GetBootnodes(nodeID)
	table, _ := kademlia.NewTable(nodeID, bootnodes, conn)
	table.NodeQueryList()
	select {}
}
