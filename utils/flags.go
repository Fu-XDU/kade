package utils

import "github.com/urfave/cli"

var (
	P2PPortFlag = cli.IntFlag{
		Name:  "port, p",
		Usage: "Network listening port",
		Value: 30303,
	}
)
