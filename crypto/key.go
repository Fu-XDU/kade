package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/labstack/gommon/log"
	"io/ioutil"
	"os"
)

func LoadKey() (sk *ecdsa.PrivateKey, pk *ecdsa.PublicKey) {
	var err error
	if !keyExists() {
		_, err = generateKeys()
		if err != nil {
			log.Fatal(err)
		}
	}

	sk, err = loadSK()
	if err != nil {
		log.Fatal(err)
	}

	pk, err = loadPK()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func loadSK() (sk *ecdsa.PrivateKey, err error) {
	skBytes, _ := ioutil.ReadFile("./static/ec-sk.pem")
	return Bytes2SK(skBytes)
}

func loadPK() (pk *ecdsa.PublicKey, err error) {
	pkBytes, _ := ioutil.ReadFile("./static/ec-pk.pem")
	return Bytes2PK(pkBytes)
}

func Bytes2SK(skBytes []byte) (sk *ecdsa.PrivateKey, err error) {
	block, _ := pem.Decode(skBytes)
	sk, err = x509.ParseECPrivateKey(block.Bytes)
	return
}

func Bytes2PK(pkBytes []byte) (pk *ecdsa.PublicKey, err error) {
	block, _ := pem.Decode(pkBytes)
	var i interface{}
	i, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return
	}
	var ok bool
	pk, ok = i.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("public key conversion err")
	}
	return
}

func generateKeys() (sk *ecdsa.PrivateKey, err error) {
	sk, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	skBytes, err := x509.MarshalECPrivateKey(sk)
	if err != nil {
		return
	}
	pkBytes, err := x509.MarshalPKIXPublicKey(sk.Public())
	if err != nil {
		return
	}
	skBlock := pem.Block{
		Type:  "ECD PRIVATE KEY",
		Bytes: skBytes,
	}
	pkBlock := pem.Block{
		Type:  "ECD PUBLIC KEY",
		Bytes: pkBytes,
	}
	skFile, _ := os.Create("./static/ec-sk.pem")
	pkFile, _ := os.Create("./static/ec-pk.pem")
	if err = pem.Encode(skFile, &skBlock); err != nil {
		return
	}
	if err = pem.Encode(pkFile, &pkBlock); err != nil {
		return
	}
	return
}

func keyExists() bool {
	_ = os.Mkdir("static", 0777)
	return exists("./static/ec-sk.pem") && exists("./static/ec-pk.pem")
}

func exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
}

func GetNodeIDFromPubKey(pub *ecdsa.PublicKey) []byte {
	return FromECDSAPub(pub)[1:]
}
