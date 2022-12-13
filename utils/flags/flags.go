package flags

import "github.com/urfave/cli"

var (
	P2PPortFlag = cli.IntFlag{
		Name:        "port, p",
		Usage:       "Network listening port",
		Value:       30303,
		Destination: &Port,
		EnvVar:      "PORT",
	}
	KeyDirFlag = cli.StringFlag{
		Name:        "keyDir, k",
		Usage:       "Key dir",
		Value:       "./static",
		Destination: &KeyDir,
		EnvVar:      "KEY_DIR",
	}
)
