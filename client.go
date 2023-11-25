package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

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

const CHUNK_SIZE = 1024    // (bytes)
const DATAGRAM_SIZE = 1096 // (bytes) 4 id + 1 type + 2 length + 1 node type + 1024 body + 64 singature

const TIMEOUT = 5 * time.Second

type node struct {
	hash     string
	children []node
}

// debug variable, set true to enable debug fmt.prints
const debug = true
const url = "/"

// var id = 0 // an incrementing counter for the id field

func main() {

	// tls stuff, its obv but i comment to annoy Mr. JC :p
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   TIMEOUT,
	}

	reader := bufio.NewReader(os.Stdin)
	menu(reader, client)
}

func menu(reader *bufio.Reader, client *http.Client) {

	// TODO prompt for the user to enter a path for their data's root
	for {
		fmt.Print(`
prot serv int!! ^_^
Options:
	list: lists existing peers
	p -a: shows p's addresses
	p -k: shows p's public key (if it has one)
	p -r: shows p's root hash
	p -d: asks for data from peer p, writes it to local file named p-timestamp
	exit: exits
=>`)

		cmd, err := reader.ReadString('\n')
		cmd = cmd[:len(cmd)-1]
		CheckErr(err, "Read err!")

		switch parseCmd(cmd) {
		case 0:
			// getPeers(client)
			DebugPrint("get peers")
		case 1:
			// peerAddr(client, peer)
			DebugPrint("addr")
		case 2:
			// peerKey(client, peer)
			DebugPrint("key")
		case 3:
			// peerRoot(client, peer)
			DebugPrint("root")
		case 4:
			// getData(peer)
			DebugPrint("data")
		case 5:
			return
		default:
			fmt.Println("Unkown command please retry ")
		}

	}

}

func parseCmd(cmd string) (ret int) {
	// TODO, code commands to ints 0-5 then return said commands + extra args if necessary
	split := strings.Split(cmd, "-")
	DebugPrint(split[0])

	switch split[0] {
	case "list":
		return 0
	case "exit":
		return 5
	default:
		switch split[1] {
		case "a":
			return 1
		case "k":
			return 2
		case "r":
			return 3
		case "d":
			return 4
		}
	}
	return -1
}

// TODO directory/file ==> merkel tree
func merkelify(path string) (root node) {
	info, err := os.Stat(path)
	CheckErr(err, "os.stat error, merkelify")
	if info.IsDir() {
		return hashDir(path)
	} else {
		return hashFile(path)
	}
}

func hashDir(path string) (root node) {
	dir, err := os.ReadDir(path)
	CheckErr(err, "os.readdir err, hashDir")
}

func hashFile(path string) (root node) {
	fi, err := os.Stat(path)
	CheckErr(err, "os.stat err, hashFile")
}

func getUrl(tcpClient *http.Client, ReqUrl string) (*http.Response, error) {

	req, err := http.NewRequest("GET", ReqUrl, nil)
	CheckErr(err, "make req")

	res, err := tcpClient.Do(req)

}

func getPeers(tcpClient *http.Client) {

	res, err := getUrl(tcpClient, url+"/peers")
	CheckErr(err, "get error /peers")
	// TODO format the addresses nicely before return
}

func peerAddr(tcpClient *http.Client, peer string) {

	res, err := getUrl(tcpClient, url+"/peers"+peer+"addresses")
	CheckErr(err, "get error /peers/p/addresses")
	// TODO jp (just print)
}

func peerKey(tcpClient *http.Client, peer string) {
	res, err := getUrl(tcpClient, url+"/peers"+peer+"addresses")
	CheckErr(err, "get error /peers/p/key")
	// TODO jp
}

func peerRoot(tcpClient *http.Client, peer string) {
	res, err := getUrl(tcpClient, url+"/peers"+peer+"addresses")
	CheckErr(err, "get error /peers/p/root")
	// TODO jp
}

func getData(peer string) {
	// TODO write to local file
}

// misc funcs
func CheckErr(err error, msg string) {
	if err != nil {
		log.Fatal(msg)
	}
}

func DebugPrint(msg interface{}) {
	if debug {
		fmt.Printf("%q\n", msg)
	}
}
