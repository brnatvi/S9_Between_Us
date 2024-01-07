package moduls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

var KeyServer *ecdsa.PublicKey = nil
var KeyPeer *ecdsa.PublicKey = nil
var MyPrivateKey *ecdsa.PrivateKey = nil
var MyPublicKey *ecdsa.PublicKey = nil

// Generate keys
func GenerateKeys() {
	MyPrivateKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	MyPublicKey, _ = MyPrivateKey.Public().(*ecdsa.PublicKey)
}

// public key to 64 bytes array
func FormatPublicKey(publicKey *ecdsa.PublicKey) []byte {
	formatted := make([]byte, 64)
	publicKey.X.FillBytes(formatted[:32])
	publicKey.Y.FillBytes(formatted[32:])
	return formatted
}

// Parse public key (in form of 64 bytes array)
func ParcePublicKay(data []byte) *ecdsa.PublicKey {
	var x, y big.Int
	if len(data) != 0 {
		x.SetBytes(data[:32])
		y.SetBytes(data[32:])
	} else {
		zeroArr := make([]byte, 64)
		x.SetBytes(zeroArr[:32])
		y.SetBytes(zeroArr[32:])
	}
	var publicKey ecdsa.PublicKey
	publicKey = ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     &x,
		Y:     &y,
	}
	return &publicKey
}

// Sign the message
func SignMessage(data []byte, privateKey *ecdsa.PrivateKey) []byte {
	hashed := sha256.Sum256(data)
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	return signature
}

// Check the signature of message
func CheckSignature(data []byte, signature []byte, publicKey *ecdsa.PublicKey) bool {
	var r, s big.Int
	r.SetBytes(signature[:32])
	s.SetBytes(signature[32:])
	hashed := sha256.Sum256(data)
	ok := ecdsa.Verify(publicKey, hashed[:], &r, &s)
	return ok
}
