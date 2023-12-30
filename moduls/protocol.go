package moduls

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const url = "https://jch.irif.fr:8443"
const TIMEOUT = 5 * time.Second
const LOG_PRINT_DATA = false

var messCounter uint32 = 1

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

const (
	RESULT_OK    = 0
	RESULT_ERROR = 1
)

// NODE TYPES (first byte of body)
const (
	CHUNK        = 0 //to rename to NODE_
	BIG_FILE     = 1 //to rename to NODE_
	DIRECTORY    = 2 //to rename to NODE_
	NODE_UNKNOWN = 3 //
)

// SIZES (bytes, count)
const (
	CHUNK_SIZE    = 1024
	DATAGRAM_SIZE = 2048 // 4 id + 1 type + 2 length + 1 node type + 1024 body + 64 singature
	ID_SIZE       = 4
	TYPE_SIZE     = 1
	LENGTH_SIZE   = 2
	HASH_SIZE     = 32
	NAME_SIZE     = 32
	VALUE_SIZE    = 32
	SIGN_SIZE     = 64
	POS_TYPE      = 4
	POS_LENGTH    = 5
	POS_HASH      = 7
	POS_VALUE     = 39
	POS_SIGN      = 71
	MAX_CHILDREN  = 32 // max children in the tree
)

// Structure for temporary storage of data downloaded from hash (chunks, big_files or directories)
type StrObject struct {
	Type   int    // can be CHUNK, BIG_FILE or -1 for all content of DIRECTORY (because the type of data is unknown yet)
	Name   string // "" for CHUNK and BIG_FILE
	NbHash int    // number of hashes, 1 for CHUNK and content of DIRECTORY, > 1 for BIG_FILE
	Hash   []byte // hash of data
	Data   []byte // data, nil for BIG_FILE and content of DIRECTORY
}

type DataObject struct {
	Type   int    // can be CHUNK, BIG_FILE or -1 for all content of DIRECTORY (because the type of data is unknown yet)
	Name   string // "" for CHUNK and BIG_FILE
	Path   string
	Handle *os.File
}

// ==========================   Main functions ========================== //

// getDatum req
func GetData(peer string) {
	//rootPeerServ := moduls.PeerRoot(client, "jch.irif.fr")
	////keyPeerServ := moduls.PeerKey(client, "jch.irif.fr")		// doesn't return a key
	//
	//fmt.Printf("peer root : %v \n", rootPeerServ)
	//fmt.Printf("peer key : %v \n", servPublicKey)
	//
	//moduls.DownloadData(conn, rootPeerServ, myPeer, "", "")
}

func SendData() {
	// TODO send requested data func
}

// Register on the server
// Parameters:
// - conn - UDP Connection
// - myPeer - name of my peer
// Return: public key of Server
func RegistrationOnServer(conn *net.UDPConn, myPeer string) []byte {

	// send Hello till reception of good HelloReply
	for {
		b, err := sendHello(conn, myPeer)
		HandlePanicError(err, "RegistrationOnServer")
		if errors.Is(err, os.ErrDeadlineExceeded) {
			PanicMessage("The respondent has stopped sending messages to your address. You need to restart registration\n")
			return nil
		}
		if b {
			break
		} else { // Another attempts to re-send HELLO after 5 sec
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

	ServerPublicKey := buf[7:l]

	// send PublicKeyReply
	buf = composeHandChakeMessage(newMessId, byte(PUBLIC_KEY_REPLY), myPeer, 0, 0)
	_, err = conn.Write(buf)
	if err != nil {
		log.Panic("PublicKeyReply: Write PUBLIC_KEY_REPLY to UDP failure\n")
		return nil
	}
	messCounter++

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

	// send Hash("")
	if !hasRoot {
		buf = composeDataSendMessage(newMessId, byte(ROOT_REPLY), 32, "")
	} else {
		// TODO
	}

	_, err = conn.Write(buf)
	if err != nil {
		log.Panic("PublicKeyReply: Write ROOT_REPLY to UDP failure\n")
		return nil
	}
	messCounter++
	isCanceled = false
	return ServerPublicKey
}

// Maintain connection with server - sends messages every 30 seconds
func MaintainConnectionServer(conn *net.UDPConn, myPeer string) {
	for {
		timeStart := time.Now()

		if !isCanceled {
			_, err := sendHello(conn, myPeer)
			HandlePanicError(err, "MaintainConnectionServer")

			if errors.Is(err, os.ErrDeadlineExceeded) {
				PanicMessage("The respondent has stopped sending messages to your address. You need to restart registration\n")
			}
			timeNow := time.Now()

			if timeStart.Sub(timeNow) >= 180*time.Second {
				isCanceled = true
				continue
			} else {
				time.Sleep(30 * time.Second)
			}

		} else {
			fmt.Printf("Connection was lost, will try to reconnect ...\n\n")
			RegistrationOnServer(conn, myPeer)
			time.Sleep(30 * time.Second)
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
// Return: list of addresses of peer
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

// Print all peer's names and adresses
func GetAllPeersAdresses(tcpClient *http.Client) {
	peersNames := GetPeers(tcpClient)
	if peersNames != nil {
		for ind, name := range peersNames {
			peerAddr := PeerAddr(tcpClient, name)
			if peerAddr != nil {
				for _, ad := range peerAddr {
					//	peersAdresses = append(peersAdresses, ad)
					fmt.Printf("%d peer : %s  has adresse : %s \n", ind, name, ad)
				}
			}
		}
		fmt.Println("")
	} else {
		fmt.Printf("Has not peers \n")
	}
}

// Get peer's key
// Obtaining the following status codes is possible:
// - 200 if the peer is known and has announced a public key, and then the body contains the key (a	sequence of 64 bytes);
// - 204 if the peer is known, but has not announced a public key;
// - 404 if the peer is not known.
// Return: key of peer
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
// Return: root of peer
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
func sendHello(conn *net.UDPConn, myPeer string) (bool, error) {

	// send HELLO
	buf := composeHandChakeMessage(messCounter, byte(HELLO), myPeer, len(myPeer)+4, 0)
	_, err := conn.Write(buf)
	if err != nil {
		HandleFatalError(err, "sendHello: Write to UDP failure")
		return false, err
	}

	bufRes := make([]byte, DATAGRAM_SIZE)
	timeStart := time.Now()

	// try to receive HELLO_REPLY until a response HELLO_REPLY with the required ID will be received till TIMEOUT
	for {
		conn.SetReadDeadline(time.Now().Add(TIMEOUT)) // set Timeout

		//recieve HELLO_REPLY
		l, _, err := conn.ReadFromUDP(bufRes)
		if err != nil {
			if err != io.EOF {
				//	HandleFatalError(err, "sendHello: ReadFromUDP error")
				messCounter++
				return false, err
			}
		}

		bExit := false
		rezCheck := CheckUDPIncomingPacket(bufRes, l, HELLO_REPLY, "HELLO_REPLY")
		switch rezCheck {
		case 0:
			bExit = true
		case 1: // exit from function to re-send HELLO
			messCounter++
			return false, errors.New("sendHello: The lenght of HELLO_REPLY recieved != expected one")

		case 2: // reject and try to pull out the next response until TIMEOUT
			timeNow := time.Now()

			// if TIMEOUT -> exit from function to re-send HELLO
			if timeStart.Sub(timeNow) >= TIMEOUT {
				messCounter++
				return false, errors.New("sendHello: Timeout reception of HELLO_REPLY")
			} else {
				continue
			}
		case 3: // exit from function to re-send HELLO
			messCounter++
			return false, errors.New("sendHello: Id HELLO_REPLY != Id HELLO")
		}

		if bExit {
			break
		}
	}
	messCounter++
	return true, nil
}

// Check incoming UDP datagramme by 3 parameters: length, type and id
// Return:
// 1 if length does not match expected length
// 2 if type does not match expected type
// 3 if id does not match expected id
// 0 if all is ok
func CheckUDPIncomingPacket(bufRes []byte, lenRecieved int, typeExp int, strTypeExp string) int {

	// check lenght  -> if error, exit from function to re-send request
	hasToBe := binary.BigEndian.Uint16(bufRes[5:7]) + 4 + 1 + 2 + 64
	fmt.Println("\n-------- check lenght -------------")
	fmt.Printf("readed    : %d\n", lenRecieved)
	fmt.Printf("has to be : %d\n", hasToBe)
	fmt.Println("---------------------\n")

	if hasToBe != uint16(lenRecieved) {
		fmt.Printf("The lenght of %s recieved != expected one", strTypeExp)
		return 1
	}

	// check type -> if error, reject and wait the next response
	if CheckTypeEquality(byte(typeExp), bufRes) == -1 {
		fmt.Printf("Not type %d was recieved, but %d\n", typeExp, bufRes[4:5][0])
		return 2
	}

	// check id  -> if error, exit from function to re-send request
	id := binary.BigEndian.Uint32(bufRes[:4])
	fmt.Println("\n---------- check id -----------")
	fmt.Printf("REQUEST  id : %v\n", messCounter)
	fmt.Printf("RESPONSE id : %v\n", id)
	fmt.Printf("typeMess  %v\n", bufRes[4:5])
	fmt.Printf("lenMess   %v\n", bufRes[5:7])
	fmt.Printf("response  %v\n\n", string(bufRes[7:7+binary.BigEndian.Uint16(bufRes[5:7])]))
	fmt.Println("---------------------\n")

	if id != messCounter {
		fmt.Printf("Id of request %d != id of response %d\n", messCounter, id)
		return 3
	}
	return 0
}

// Send "GetDatum" & Recieve "Datum"
// Return: list of strObjects
func GetDataByHash(conn *net.UDPConn, hash []byte, myPeer string) ([]byte, error) {

	fmt.Printf(">GetDataByHash(..., %v..., %s)\n", hash[0:32], myPeer)

	isRecieved := false

	// send GetDatum
	buf := composeGetDatumMessage(messCounter, byte(GET_DATUM), myPeer, HASH_SIZE, hash, 0)
	_, err := conn.Write(buf)
	if err != nil {
		PanicMessage("GetDataByHash: Write to UDP failure\n")
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(TIMEOUT)) // set Timeout

	bufRes := make([]byte, DATAGRAM_SIZE)
	timeStart := time.Now()

	// receive Datum until a response with the required ID is received
	for !isRecieved {

		// receive Datum
		all, _, err := conn.ReadFromUDP(bufRes)
		if err != nil {
			if err != io.EOF {
				messCounter++
				fmt.Printf("GetDataByHash: ReadFromUDP error %v\n", err)
				return nil, err
			}
		}

		// check id
		if binary.BigEndian.Uint32(bufRes[:POS_TYPE]) == messCounter {
			// check type
			if CheckTypeEquality(byte(DATUM), bufRes) == -1 {
				if CheckTypeEquality(byte(NO_DATUM), bufRes) == -1 {
					messCounter++
					fmt.Printf("GetDataByHash: neither DATUM nor NO_DATUM was received\n")
					return nil, nil
				} else {
					messCounter++
					fmt.Printf("GetDataByHash: NO_DATUM was received %d\n", messCounter)
					return nil, NoDatumRecieved()
				}
			}

			fmt.Printf("Was recieved %d bytes at all\n", all)
			fmt.Printf("Length     = %d bytes \n", binary.BigEndian.Uint16(bufRes[POS_LENGTH:POS_HASH]))

			// find lenght of value = Length - HASH_SIZE
			lenValue := binary.BigEndian.Uint16(bufRes[POS_LENGTH:POS_HASH]) - HASH_SIZE

			fmt.Printf("Was recieved %d bytes of value\n", lenValue)

			// Check hash 1 : if hash in GetDatum == hash in DATUM
			for i := 0; i < HASH_SIZE; i++ {
				if hash[i] != bufRes[POS_HASH+i] {
					messCounter++
					PanicMessage("GetDataByHash: Data substitution !!! The hash I received is not the one I've asked for\n")
					return nil, nil
				}
			}

			value := bufRes[POS_VALUE:(POS_VALUE + lenValue)]

			// Check hash 2 : if hash in DATUM is really hash of value (there was no value substitution)
			hashedValue := sha256.Sum256(value)

			for i := 0; i < HASH_SIZE; i++ {
				if hashedValue[i] != hash[i] {
					messCounter++
					PanicMessage("GetDataByHash: Data substitution !!! The hash(value) does not match the one I've asked for\n")
					return nil, nil
				}
			}

			messCounter++
			isRecieved = true

			if LOG_PRINT_DATA {
				fmt.Printf("GetDataByHash Value: %v \n\n", value)
			}

			return value, nil

		} else {
			fmt.Printf("GetDataByHash: MessageId DATUM != MessageId GET_DATUM\n")
			timeNow := time.Now()

			if timeStart.Sub(timeNow) >= TIMEOUT {
				messCounter++
				PanicMessage("GetDataByHash: Timeout reception of DATUM\n")
				return nil, nil
			}
		}
	}
	return nil, nil
}

// Parser for data obtained by hash.
// (See the principles of filling the structure above, in the description of the structure)
// Parameter: binary array to parce
// Returns: a list of StrObjects whose length = 1 for CHUNK and BIG_FILE, >=1 for DIRECTORY
func ParceValue(data []byte) []StrObject {

	fmt.Printf(">ParceValue(...)\n")

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
		newObj.NbHash = 1
		newObj.Data = data[point:l]

		listContent = append(listContent, newObj)
		return listContent

	case BIG_FILE:
		var newObj StrObject
		newObj.Type = BIG_FILE
		newObj.Name = ""
		newObj.Data = nil
		newObj.NbHash = (l - 1) / HASH_SIZE
		fmt.Printf("Hash count of BIG_FILE : %d\n", newObj.NbHash)

		// Bring together all the hashes of a large file
		newObj.Hash = append(newObj.Hash, data[point:point+(newObj.NbHash*HASH_SIZE)]...)

		listContent = append(listContent, newObj)
		return listContent

	case DIRECTORY:
		// Directory contains a list of elements of the forme: name(32 bytes) + hash(32 bytes)
		for point < (l - 1) {
			var newObj StrObject
			var binName []byte

			binNameZeros := data[point:(point + NAME_SIZE)]

			for _, b := range binNameZeros {
				if b != byte(0) {
					binName = append(binName, b)
				}
			}
			newObj.Name = string(binName)
			point = point + NAME_SIZE

			newObj.Hash = data[point:(point + HASH_SIZE)]
			point = point + HASH_SIZE

			fmt.Printf("newObj.Name : %s, newObj.Hash : %v\n", newObj.Name, newObj.Hash)

			// their types are still unknown, so NODE_UNKNOWN
			newObj.Type = NODE_UNKNOWN
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

// Download data from hash and create directory structure (files and folders)
// Recursive function ! Used to download data to the depth of the directory structure.
// So, the first call is made with arguments  nameData = "" and parentName = ""
// Parameters:
// - conn - UDP Connection
// - hashPeer - hash of peer
// - myPeer - name of my peer
// - DataObj - data object, holds information about current file and diectory
func DownloadData(conn *net.UDPConn, hashPeer []byte, myPeer string, DataObj *DataObject) int {

	fmt.Printf(">DownloadData(..., %v..., %s, %s, %s)\n", hashPeer[0:32], myPeer, DataObj.Name, DataObj.Path)

	value, _ := GetDataByHash(conn, hashPeer, myPeer)

	if len(value) != 0 {
		if LOG_PRINT_DATA {
			fmt.Printf("DownloadData: Value from GetDataByHash : %v \n", value)
		}

		// Parcing the value recieved
		var listContent []StrObject
		listContent = ParceValue(value)

		// Create the files and directories
		for _, el := range listContent {
			if el.Type == CHUNK {
				fmt.Printf("DownloadData: CHUNK for file %s\n", DataObj.Name)
				if DataObj.Handle == nil {
					FilePath := DataObj.Path
					FilePath = filepath.Join(FilePath, DataObj.Name)
					hndl, err := os.OpenFile(FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
					DataObj.Handle = hndl
					if err != nil {
						log.Fatal(err)
						return RESULT_ERROR
					}
				}

				if _, err := DataObj.Handle.Write(el.Data); err != nil {
					DataObj.Handle.Close()
					DataObj.Handle = nil
					log.Fatal(err)
					return RESULT_ERROR
				}
			} else if el.Type == BIG_FILE {
				fmt.Printf("DownloadData: BIG_FILE for file %s\n", DataObj.Name)

				point := 0
				fmt.Printf("Len of BIG_FILE bufer\n")

				// The ParceValue function brought together all the hashes of a large file
				// So to receive data, we need to send requests for each 32 byte pieces:
				for i := 0; i < el.NbHash; i++ {
					res := DownloadData(conn, el.Hash[point:point+HASH_SIZE], myPeer, DataObj)
					point = point + HASH_SIZE
					if res != RESULT_OK {
						return res
					}
				}
			} else if el.Type == NODE_UNKNOWN {

				Path := DataObj.Path
				Path = filepath.Join(Path, DataObj.Name)

				if _, err := os.Stat(Path); os.IsNotExist(err) {
					os.Mkdir(Path, 0777)
				}

				fmt.Println("=============================================================")
				fmt.Printf("DownloadData: DIR for content %s of directory %s\n", el.Name, Path)

				fmt.Printf("Name : %s, Hash : %v\n", el.Name, el.Hash)

				ChildObj := DataObject{NODE_UNKNOWN, el.Name, Path, nil}
				// recursive call
				res := DownloadData(conn, el.Hash, myPeer, &ChildObj)
				if res != RESULT_OK {
					return res
				}

				if ChildObj.Handle != nil {
					err := ChildObj.Handle.Close()
					if err != nil {
						log.Fatal(err)
					}
				}
			} else {
				// do nothing
			}
		}
	}
	return RESULT_OK
}
