package moduls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
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

const url = "https://jch.irif.fr:8443/"
const TIMEOUT = 5 * time.Second

var messCounter uint32 = 1

const name string = "5miles"

const CHUNK_SIZE = 1024    // (bytes)
const DATAGRAM_SIZE = 1096 // (bytes) 4 id + 1 type + 2 length + 1 node type + 1024 body + 64 singature

var isCanceled bool = false // if need to maintain connection with server

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
	CHUNK     = 0
	BIG_FILE  = 1
	DIRECTORY = 2
)

func RegistrationOnServer(tcpClient *http.Client) {

	// send Hello
	res, err := sendHello(tcpClient)
	HandlePanicError(err, "sendHello failure")

	//recieve HelloReply
	body, err := ioutil.ReadAll(res.Body)
	HandlePanicError(err, "ReadAll failure")
	defer res.Body.Close()

	fmt.Println(string(body))

	//recieve PublicKey

	// send PublicKeyReply

	// recieve Root

	// send Hash("")

}

func sendHello(tcpClient *http.Client) (*http.Response, error) {

	buf := composeHandChakeMessage(messCounter, byte(HELLO), len(name)+4, 0)

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewReader(buf))
	HandlePanicError(err, "NewRequest failure")

	body, err := ioutil.ReadAll(req.Body)
	fmt.Printf("req body   : %v\n", body)
	defer req.Body.Close()

	req.Header.Add("Content-Type", "application/octet-stream")

	return tcpClient.Do(req)
}

func composeHandChakeMessage(idMes uint32, typeMes uint8, lenMes int, extentMes int) []byte {

	var buf bytes.Buffer

	i := make([]byte, 4)
	binary.BigEndian.PutUint32(i, idMes)
	buf.Write(i)
	fmt.Printf("%v\n", buf.Bytes())

	buf.WriteByte(typeMes)
	fmt.Printf("%v\n", buf.Bytes())

	j := make([]byte, 2)
	binary.BigEndian.PutUint16(j, uint16(lenMes))
	buf.Write(j)
	fmt.Printf("%v\n", buf.Bytes())

	k := make([]byte, 4)
	binary.BigEndian.PutUint32(k, uint32(extentMes))
	buf.Write(k)
	fmt.Printf("%v\n", buf.Bytes())

	buf.WriteString(name)
	fmt.Printf("%v\n", buf.Bytes())

	return buf.Bytes()
}

func MaintainConnectionServer(tcpClient *http.Client) {
	if !isCanceled {

		// send Hello
		res, err := sendHello(tcpClient)
		HandlePanicError(err, "sendHello failure")

		//recieve HelloReply
		body, err := ioutil.ReadAll(res.Body)
		HandlePanicError(err, "ReadAll failure")
		defer res.Body.Close()

		fmt.Println(string(body))

		// TODO Handle absence of response

		time.Sleep(30 * time.Second)
	}
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
