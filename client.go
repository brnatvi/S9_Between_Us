package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"client.go/moduls"
)

// var id = 0 // an incrementing counter for the id field

const TIMEOUT = 5 * time.Second

func main() {

	myPeer := os.Args[1]

	// init params and create merkel tree
	//	name, port, dirPath := readConfig("config")
	//
	//	var root moduls.Node
	//
	//	if len(dirPath) != 0 {
	//		root = moduls.Merkelify(dirPath)
	//	}

	// addresses of server and peers
	var servAdresses []string
	var servPublicKey []byte

	// Create TCP client
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   TIMEOUT,
	}

	// Get addresses of server
	servAdresses = GetServerAdresses(client)

	// Create UDP connection with server
	addr, err := net.ResolveUDPAddr("udp", servAdresses[0])
	moduls.HandleFatalError(err, "ResolveUDPAddr failure")

	conn, err := net.DialUDP("udp", nil, addr)
	moduls.HandleFatalError(err, "DialUDP failure")

	// Register on Server
	servPublicKey = moduls.RegistrationOnServer(conn, myPeer)

	rootPeerServ := moduls.PeerRoot(client, "jch.irif.fr")

	fmt.Printf("peer root : %v \n", rootPeerServ)
	fmt.Printf("peer key : %v \n", servPublicKey)

	DataObj := moduls.DataObject{moduls.NODE_UNKNOWN, "", "", nil}
	Path, err := os.Getwd()

	if err != nil {
		fmt.Printf("Panic\n")
		return
	}

	DataObj.Path = Path
	DataObj.Path = filepath.Join(DataObj.Path, "Recieved_Data")
	if _, err := os.Stat(DataObj.Path); os.IsNotExist(err) {
		os.Mkdir(DataObj.Path, 0777)
	}

	moduls.DownloadData(conn, rootPeerServ, myPeer, &DataObj)

	//moduls.MaintainConnectionServer(conn, myPeer)

	//	reader := bufio.NewReader(os.Stdin)
	//	menu(reader, client)
}

// name says it all
func readConfig(filename string) (name string, port string, dirPath string) {

	name, port, dirPath = "", "", ""

	file, err := os.Open(filename)
	moduls.HandlePanicError(err, "error opening config file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, "=")

		if len(splitLine) != 2 {
			continue
		}

		switch splitLine[0] {

		case "name":
			name = splitLine[1]
		case "port":
			port = splitLine[1]
		case "path":
			dirPath = splitLine[1]

		}
	}

	if len(name) == 0 || len(port) == 0 {
		moduls.PanicMessage("missing params in config file")
	}

	return name, port, dirPath
}

func menu(reader *bufio.Reader, client *http.Client) {

	// TODO prompt for username
	// TODO prompt for the user to enter a path for their data's root
	// TODO p -d interactions(?) after first request
	for {
		fmt.Print(`
prot serv int!! ^_^
Options:
	list: lists existing peers
	p -a: shows p's addresses
	p -k: shows p's public key (if it has one)
	p -r: shows p's root hash
	p -d: asks for data from peer p, writes it to local file named p-timestamp
	files: (on hold)
	exit: exits
=>`)
		cmd, err := reader.ReadString('\n')
		cmd = cmd[:len(cmd)-1]
		moduls.HandlePanicError(err, "Read err!")

		command, peer := parseCmd(cmd)
		switch command {
		case 0:
			moduls.GetAllPeersAdresses(client)
			moduls.DebugPrint("get peers")
		case 1:
			moduls.PeerAddr(client, peer)
			moduls.DebugPrint("addr")
		case 2:
			moduls.PeerKey(client, peer)
			moduls.DebugPrint("key")
		case 3:
			moduls.PeerRoot(client, peer)
			moduls.DebugPrint("root")
		case 4:
			//	moduls.GetData(peer)
			moduls.DebugPrint("data")
		case 5:
			return
		default:
			fmt.Println("Unkown command please retry ")
		}
	}
}

func parseCmd(cmd string) (ret int, peer string) {
	// TODO, code commands to ints 0-5 then return said commands + extra args if necessary
	split := strings.Split(cmd, "-")
	split[0] = strings.TrimSpace(split[0])
	moduls.DebugPrint(split[0])
	switch split[0] {
	case "list":
		return 0, ""
	case "exit":
		return 5, ""
	default:
		switch split[1] {
		case "a":
			return 1, split[0]
		case "k":
			return 2, split[0]
		case "r":
			return 3, split[0]
		case "d":
			return 4, split[0]
		}
	}
	return -1, ""
}

// Get addresses of server
func GetServerAdresses(tcpClient *http.Client) []string {
	res, _ := moduls.SendGetRequest(tcpClient, "https://jch.irif.fr:8443/peers/jch.irif.fr/addresses")
	if res.StatusCode == 200 {
		var servAdresses []string

		body, _ := io.ReadAll(res.Body)
		res.Body.Close()

		strBody := string(body[:])
		adresses := strings.Split(strBody, "\n")

		for _, addr := range adresses {
			if addr != "" {
				servAdresses = append(servAdresses, addr)
			}
		}
		return servAdresses
	} else {
		fmt.Printf("GetRequest of servers' addresses returned with StatusCode = %d\n", res.StatusCode)
		return nil
	}
}
