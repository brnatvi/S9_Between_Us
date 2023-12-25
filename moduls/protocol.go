package moduls

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type message struct {
	id     []byte
	type_  []byte
	length []byte
	body   []byte
}

// TODO get from debug

const url = "https://jch.irif.fr:8443"
const TIMEOUT = 5 * time.Second

var messCounter uint32 = 1

const name string = "5miles"

const CHUNK_SIZE = 1024    // (bytes)
const DATAGRAM_SIZE = 1096 // (bytes) 4 id + 1 type + 2 length + 1 node type + 1024 body + 64 singature

var isCanceled bool = true // if need to maintain connection with server
var hasRoot bool = false   // if we have a tree

// MESSAGE TYPES
const (
	NO_OP                 = 0
	ERROR                 = 1
	ERROR_REPLY           = 128
	HELLO                 = 2
	HELLO_REPLY           = 129
	PUBLIC_KEY            = 3
	PUBLIC_KEY_REPLY      = 130
	ROOT                  = 4
	ROOT_REPLY            = 131
	GET_DATUM             = 5
	DATUM                 = 132
	NO_DATUM              = 133
	NAT_TRAVERSAL_REQUEST = 6
	NAT_TRAVERSAL         = 7
)

// NODE TYPES (first byte of body)
const (
	CHUNK     = 0 // 0
	BIG_FILE  = 1 // 1
	DIRECTORY = 2 // 2
	// this one is just for the max children in the tree
	MAX_CHILDREN = 32 // 32
)

// Register on the server
func RegistrationOnServer(conn *net.UDPConn) {

	isRecieved := false

	// send Hello till reception HelloReply
	for !isRecieved {
		err := sendHello(conn)
		if err == nil {
			isRecieved = true
		} else {
			time.Sleep(TIMEOUT)
		}
	}

	//recieve PublicKey
	buf := make([]byte, DATAGRAM_SIZE)
	l, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("PublicKey: ReadFromUDP error %v\n", err)
			return
		}
	}
	if checkIfErrorRecieved(byte(PUBLIC_KEY), buf) == -1 {
		return
	}

	newMessId := binary.BigEndian.Uint32(buf[:4])
	fmt.Printf("newMessId %d\n", newMessId)

	ServerPublicKey := buf[7:l]
	fmt.Printf("PublicKey : %v\n", ServerPublicKey)

	// send PublicKeyReply
	buf = composeHandChakeMessage(newMessId, byte(PUBLIC_KEY_REPLY), 0, 0)
	_, err = conn.Write(buf)
	if err != nil {
		log.Panic("PublicKeyReply: Write to UDP failure\n")
		return
	}

	// recieve Root
	buf = make([]byte, DATAGRAM_SIZE)
	l, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Root: ReadFromUDP error %v\n", err)
			return
		}
	}
	if checkIfErrorRecieved(byte(ROOT), buf) == -1 {
		return
	}
	newMessId = binary.BigEndian.Uint32(buf[:4])
	fmt.Printf("newMessId %d\n", newMessId)

	// send Hash("")
	if !hasRoot {
		composeDataSendMessage(newMessId, byte(ROOT_REPLY), 32, "")
	} else {
		// TODO
	}
	messCounter++
}

// Maintain connection with server - sends messages every 30 seconds
func MaintainConnectionServer(conn *net.UDPConn) {
	for {
		if isCanceled {

			err := sendHello(conn)
			HandlePanicError(err, "sendHello failure")
			messCounter++

			// TODO Handle absence of response

			time.Sleep(30 * time.Second)

		} else {
			fmt.Printf("Connection was lost, will try to reconnect ...\n\n")
			RegistrationOnServer(conn)
		}
	}
}

// ==========================   Auxiliary functions ========================== //

// Send "Hello" & Recieve "HelloReply"
func sendHello(conn *net.UDPConn) error {
	// send Hello
	buf := composeHandChakeMessage(messCounter, byte(HELLO), len(name)+4, 0)
	_, err := conn.Write(buf)
	if err != nil {
		log.Panic("Write to UDP failure\n")
		return err
	}

	//recieve HelloReply
	bufRes := make([]byte, DATAGRAM_SIZE)
	l, _, err := conn.ReadFromUDP(bufRes)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("ReadFromUDP error %v\n", err)
			return err
		}
	}
	if binary.BigEndian.Uint32(bufRes[:4]) != messCounter {
		fmt.Printf("MessageId HelloReply server's != My MessageId Hello send\n")
		// TODO Heandler
		return nil
	}
	fmt.Printf("idMessage %v\n", bufRes[0:4])
	fmt.Printf("typeMess  %v\n", bufRes[4:5])
	fmt.Printf("lenMess   %v\n", bufRes[5:7])
	fmt.Printf("response  %v\n\n", string(bufRes[7:l]))

	if isCanceled {
		if checkIfErrorRecieved(byte(HELLO_REPLY), bufRes) == -1 {
			return err
		}
	} else {
		if checkIfErrorRecieved(byte(HELLO), buf) == -1 {
			return err
		}
	}

	isCanceled = false
	return err
}

// Compose UDP handshake message (with a peer or server) and convert it to binary
func composeHandChakeMessage(idMes uint32, typeMes uint8, lenMes int, extentMes int) []byte {

	var buf bytes.Buffer

	i := make([]byte, 4)
	binary.BigEndian.PutUint32(i, idMes)
	buf.Write(i)

	buf.WriteByte(typeMes)

	j := make([]byte, 2)
	binary.BigEndian.PutUint16(j, uint16(lenMes))
	buf.Write(j)

	k := make([]byte, 4)
	binary.BigEndian.PutUint32(k, uint32(extentMes))
	buf.Write(k)

	buf.WriteString(name)
	fmt.Printf("my bin message : %v\n\n", buf.Bytes()) // for debug

	return buf.Bytes()
}

// Composes UDP message to send data and converts it to binary
func composeDataSendMessage(idMes uint32, typeMes uint8, lenMes int, valueMes string) []byte {

	var buf bytes.Buffer

	i := make([]byte, 4)
	binary.BigEndian.PutUint32(i, idMes)
	buf.Write(i)

	buf.WriteByte(typeMes)

	j := make([]byte, 2)
	binary.BigEndian.PutUint16(j, uint16(lenMes))
	buf.Write(j)

	hash := sha256.Sum256([]byte(valueMes))
	buf.Write(hash[:])

	buf.WriteString(valueMes)
	fmt.Printf("my bin message : %v\n\n", buf.Bytes()) // for debug

	return buf.Bytes()
}

func SendGetRequest(tcpClient *http.Client, ReqUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", ReqUrl, nil)
	HandlePanicError(err, "NewRequest failure")

	res, err := tcpClient.Do(req)

	return res, err
}

func GetPeers(tcpClient *http.Client) {
	res, err := SendGetRequest(tcpClient, url+"peers")
	HandlePanicError(err, "get error /peers")
	// TODO Sormat the addresses nicely before return
	DebugPrint(res.Body)
}

func PeerAddr(tcpClient *http.Client, peer string) {

	res, err := SendGetRequest(tcpClient, url+"peers/"+peer+"/addresses")
	DebugPrint(url + "peers/" + peer + "/addresses")
	HandlePanicError(err, "get error /peers/p/addresses")
	// TODO jp (just print)
	p := make([]byte, 200)
	res.Body.Read(p)
	DebugPrint(p)

}

func PeerKey(tcpClient *http.Client, peer string) {
	res, err := SendGetRequest(tcpClient, url+"peers/"+peer+"/key")
	HandlePanicError(err, "get error /peers/p/key")
	// TODO Sp
	DebugPrint(res.Body)

}

func PeerRoot(tcpClient *http.Client, peer string) {
	res, err := SendGetRequest(tcpClient, url+"peers/"+peer+"/root")
	HandlePanicError(err, "get error /peers/p/root")
	// TODO Sp
	DebugPrint(res.Body)
}

// getDatum req
func GetData(peer string) {
	// TODO write to local file
}

func SendData() {
	// TODO send requested data func
}
