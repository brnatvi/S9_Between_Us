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
	"strings"
	"time"
)

type StrObject struct {
	Type   int
	Name   string
	NbHash int
	Hash   []byte
	Data   []byte
}

// TODO get from debug

const url = "https://jch.irif.fr:8443"
const TIMEOUT = 5 * time.Second

var messCounter uint32 = 1

//const name string = "5miles"

const CHUNK_SIZE = 1024    // (bytes)
const DATAGRAM_SIZE = 2048 // (bytes) 4 id + 1 type + 2 length + 1 node type + 1024 body + 64 singature

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

// ==========================   Main functions ========================== //

// getDatum req
func GetData(peer string) {
	// TODO write to local file
}

func SendData() {
	// TODO send requested data func
}

// Register on the server
func RegistrationOnServer(conn *net.UDPConn, myPeer string) []byte {

	isRecieved := false

	// send Hello till reception HelloReply
	for !isRecieved {
		err := sendHello(conn, myPeer)
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
			return nil
		}
	}
	if CheckTypeEquality(byte(PUBLIC_KEY), buf) == -1 {
		return nil
	}

	newMessId := binary.BigEndian.Uint32(buf[:4])
	fmt.Printf("newMessId %d\n", newMessId)

	ServerPublicKey := buf[7:l]
	fmt.Printf("PublicKey : %v\n", ServerPublicKey)

	// send PublicKeyReply
	buf = composeHandChakeMessage(newMessId, byte(PUBLIC_KEY_REPLY), myPeer, 0, 0)
	_, err = conn.Write(buf)
	if err != nil {
		log.Panic("PublicKeyReply: Write to UDP failure\n")
		return nil
	}

	// recieve Root
	buf = make([]byte, DATAGRAM_SIZE)
	l, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Printf("Root: ReadFromUDP error %v\n", err)
			return nil
		}
	}
	if CheckTypeEquality(byte(ROOT), buf) == -1 {
		return nil
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
	return ServerPublicKey
}

// Maintain connection with server - sends messages every 30 seconds
func MaintainConnectionServer(conn *net.UDPConn, myPeer string) {
	for {
		if isCanceled {

			err := sendHello(conn, myPeer)
			HandlePanicError(err, "sendHello failure")
			messCounter++

			// TODO Handle absence of response

			time.Sleep(30 * time.Second)

		} else {
			fmt.Printf("Connection was lost, will try to reconnect ...\n\n")
			RegistrationOnServer(conn, myPeer)
		}
	}
}

// ==========================   Auxiliary TCP functions ========================== //

func SendGetRequest(tcpClient *http.Client, ReqUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", ReqUrl, nil)
	HandlePanicError(err, "NewRequest failure")

	res, err := tcpClient.Do(req)

	return res, err
}

// Get peers' names
func GetPeers(tcpClient *http.Client) []string {
	res, err := SendGetRequest(tcpClient, url+"/peers/")
	HandlePanicError(err, "GetPeers: Get failure")
	if err != nil {
		return nil
	}

	if res.StatusCode == 200 {
		var peersNames []string

		body, _ := io.ReadAll(res.Body)
		strBody := string(body[:])
		adresses := strings.Split(strBody, "\n")

		for _, addr := range adresses {
			if addr != "" {
				peersNames = append(peersNames, addr)
			}
		}
		res.Body.Close()
		return peersNames
	} else {
		fmt.Printf("GetPeers: GetRequest of servers' addresses returned with StatusCode = %d\n", res.StatusCode)
		res.Body.Close()
		return nil
	}
}

// Get peer's address
// Obtaining the following status codes is possible:
// - 200 if the peer is known, and then the body contains a list of UDP socket addresses, one per line;
// - 404 if peer is not known.
func PeerAddr(tcpClient *http.Client, peer string) []string {
	res, err := SendGetRequest(tcpClient, url+"/peers/"+peer+"/addresses")
	HandlePanicError(err, "PeerAddr: Get failure")
	if err != nil {
		return nil
	}

	if res.StatusCode == 404 {
		fmt.Printf("PeerAddr: Peer %s is unknown\n", peer)
		return nil
	} else {
		var peerNames []string
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()

		strBody := string(body[:])
		adresses := strings.Split(strBody, "\n")

		for _, addr := range adresses {
			if addr != "" {
				peerNames = append(peerNames, addr)
			}
		}
		return peerNames
	}
}

// Get peer's key
// Obtaining the following status codes is possible:
// - 200 if the peer is known and has announced a public key, and then the body contains the key (a	sequence of 64 bytes);
// - 204 if the peer is known, but has not announced a public key;
// - 404 if the peer is not known.
func PeerKey(tcpClient *http.Client, peer string) []byte {
	res, err := SendGetRequest(tcpClient, url+"/peers/"+peer+"/key")
	HandlePanicError(err, "PeerKey: Get failure")
	if err != nil {
		return nil
	}

	switch res.StatusCode {
	case 200:
		key := make([]byte, 200)
		res.Body.Read(key)
		res.Body.Close()
		return key
	case 404:
		fmt.Printf("PeerKey: Peer %s is unknown\n", peer)
		return nil
	case 204:
		fmt.Printf("PeerKey: Peer %s is known, but has not announced public key\n", peer)
		return nil
	default:
		fmt.Printf("PeerKey: Unexpected StatusCode %d for peer %s \n", res.StatusCode, peer)
		return nil
	}
}

// Get peer's root
// Obtaining the following status codes is possible:
// - 200 if the peer is known and announced a root, and then the body contains the root hash (a sequence of 32 bytes);
// - 204 if the peer is known, but has not announced a root to the server;
// - 404 if the peer is not known.

func PeerRoot(tcpClient *http.Client, peer string) []byte {
	res, err := SendGetRequest(tcpClient, url+"/peers/"+peer+"/root")
	HandlePanicError(err, "PeerKey: Get failure")
	if err != nil {
		return nil
	}

	switch res.StatusCode {
	case 200:
		root := make([]byte, 32)
		res.Body.Read(root)
		res.Body.Close()
		return root
	case 404:
		fmt.Printf("PeerRoot: Peer %s is unknown\n", peer)
		return nil
	case 204:
		fmt.Printf("PeerRoot: Peer %s is known, but has not announced public key\n", peer)
		return nil
	default:
		fmt.Printf("PeerRoot: Unexpected StatusCode %d for peer %s \n", res.StatusCode, peer)
		return nil
	}
}

// ==========================   Auxiliary UDP functions ========================== //

// Compose UDP handshake message (with a peer or server) and convert it to binary
func composeHandChakeMessage(idMes uint32, typeMes uint8, myPeer string, lenMes int, extentMes int) []byte {

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

	buf.WriteString(myPeer)
	//fmt.Printf("my bin message : %v\n\n", buf.Bytes()) // for debug

	return buf.Bytes()
}

// Compose UDP message GetDatum and convert it to binary
func composeGetDatumMessage(idMes uint32, typeMes uint8, myPeer string, lenMes int, hash []byte, extentMes int) []byte {

	var buf bytes.Buffer

	i := make([]byte, 4)
	binary.BigEndian.PutUint32(i, idMes)
	buf.Write(i)

	buf.WriteByte(typeMes)

	j := make([]byte, 2)
	binary.BigEndian.PutUint16(j, uint16(lenMes))
	buf.Write(j)

	buf.Write(hash)

	k := make([]byte, 4)
	binary.BigEndian.PutUint32(k, uint32(extentMes))
	buf.Write(k)

	buf.WriteString(myPeer)
	//fmt.Printf("my bin message : %v\n\n", buf.Bytes()) // for debug

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
	//fmt.Printf("my bin message : %v\n\n", buf.Bytes()) // for debug

	return buf.Bytes()
}

// Send "Hello" & Recieve "HelloReply"
func sendHello(conn *net.UDPConn, myPeer string) error {
	isRecieved := false

	// send Hello
	buf := composeHandChakeMessage(messCounter, byte(HELLO), myPeer, len(myPeer)+4, 0)
	_, err := conn.Write(buf)
	if err != nil {
		log.Panic("sendHello: Write to UDP failure\n")
		return err
	}

	bufRes := make([]byte, DATAGRAM_SIZE)
	timeStart := time.Now()

	// receive messages until a response with the required ID is received
	for !isRecieved {

		//recieve HelloReply
		l, _, err := conn.ReadFromUDP(bufRes)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("sendHello: ReadFromUDP error %v\n", err)
				return err
			}
		}

		// check id
		if binary.BigEndian.Uint32(bufRes[:4]) == messCounter {
			fmt.Printf("idMessage %v\n", bufRes[0:4])
			fmt.Printf("typeMess  %v\n", bufRes[4:5])
			fmt.Printf("lenMess   %v\n", bufRes[5:7])
			fmt.Printf("response + sign  %v\n\n", string(bufRes[7:l]))

			isRecieved = true
		} else {
			fmt.Printf("sendHello: MessageId HelloReply != MessageId Hello\n")
			timeNow := time.Now()

			if timeStart.Sub(timeNow) >= TIMEOUT {
				PanicMessage("sendHello: Timeout reception of HelloReply\n")
				return err
			}
		}
	}

	// check type
	if isCanceled {
		if CheckTypeEquality(byte(HELLO_REPLY), bufRes) == -1 {
			fmt.Printf("sendHello: Not HELLO_REPLY was recieved\n")
			return err
		}
	} else {
		if CheckTypeEquality(byte(HELLO), bufRes) == -1 {
			fmt.Printf("sendHello: Not HELLO was recieved\n")
			return err
		}
	}

	isCanceled = false
	return err
}

// Send "GetDatum" & Recieve "Datum"
func GetDataByHash(conn *net.UDPConn, hash []byte, myPeer string) []byte {

	isRecieved := false

	// send GetDatum
	buf := composeGetDatumMessage(messCounter, byte(GET_DATUM), myPeer, 32, hash, 0)
	_, err := conn.Write(buf)
	if err != nil {
		PanicMessage("GetDataByHash: Write to UDP failure\n")
		return nil
	}

	bufRes := make([]byte, DATAGRAM_SIZE)
	timeStart := time.Now()

	// receive Datum until a response with the required ID is received
	for !isRecieved {

		// receive Datum
		all, _, err := conn.ReadFromUDP(bufRes)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("GetDataByHash: ReadFromUDP error %v\n", err)
				return nil
			}
		}

		// check id
		if binary.BigEndian.Uint32(bufRes[:4]) == messCounter {
			// check type
			if CheckTypeEquality(byte(DATUM), bufRes) == -1 {
				if CheckTypeEquality(byte(NO_DATUM), bufRes) == -1 {
					fmt.Printf("GetDataByHash: neither DATUM nor NO_DATUM was received\n")
					return nil
				} else {
					fmt.Printf("GetDataByHash: NO_DATUM was received\n")
					return nil
				}
			}

			fmt.Printf("Was recieved %d bytes at all\n", all)

			lenValue := binary.BigEndian.Uint16(bufRes[5:7]) - 32

			// Check hash 1 : if hash in GetDatum == hash in DATUM
			for i := 0; i < 32; i++ {
				if bufRes[7+i] != hash[i] {
					PanicMessage("GetDataByHash: Data substitution !!! The hash I received is not the one I asked for\n")
					return nil
				}
			}

			value := bufRes[39:(39 + lenValue)]

			// Check hash 2 : if hash in DATUM is really hash of value (there was no value substitution)
			hashedValue := sha256.Sum256(value)

			for i := 0; i < 32; i++ {
				if hashedValue[i] != hash[i] {
					PanicMessage("GetDataByHash: Data substitution !!! The hash(value) does not match the one I asked for\n")
					return nil
				}
			}

			messCounter++
			isRecieved = true

			return value

		} else {
			fmt.Printf("GetDataByHash: MessageId DATUM != MessageId GET_DATUM\n")
			timeNow := time.Now()

			if timeStart.Sub(timeNow) >= TIMEOUT {
				PanicMessage("GetDataByHash: Timeout reception of DATUM\n")
				return nil
			}
		}
	}
	return nil
}

func ParceValue(data []byte) []StrObject {

	var listContent []StrObject

	l := len(data)
	point := 0

	typeObj := int(data[point:(point + 1)][0])
	point = point + 1

	switch typeObj {
	case CHUNK:
		var newObj StrObject
		newObj.Type = CHUNK
		newObj.Name = ""
		newObj.NbHash = 0
		newObj.Data = data[point:l]

		listContent = append(listContent, newObj)
		return listContent

	case BIG_FILE:
		var newObj StrObject
		newObj.Type = BIG_FILE
		newObj.Name = ""
		newObj.Data = nil

		c := 1
		for c <= 32 {
			newObj.Hash = append(newObj.Hash, data[point:(point+32)]...)
			point = point + 32
			c++
		}
		newObj.NbHash = c - 1

		listContent = append(listContent, newObj)
		return listContent

	case DIRECTORY:
		// Directory contains a list of elements of the forme: name(32 bytes) + hash(32 bytes)
		for point < l {
			var newObj StrObject
			newObj.Name = string(data[point:(point + 32)])
			point = point + 32
			newObj.Hash = data[point:(point + 32)]
			point = point + 32

			// their types are still unknown, so -1
			newObj.Type = -1
			newObj.NbHash = 1
			newObj.Data = nil
			listContent = append(listContent, newObj)
		}
		return listContent

	default:
		fmt.Printf("ParceValue: Unexpected type of data recieved\n")
		return nil
	}
}
