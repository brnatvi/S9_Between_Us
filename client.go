package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"client.go/moduls"
)

// var id = 0 // an incrementing counter for the id field

const TIMEOUT = 5 * time.Second

func main() {

	// TODO initialisation from config

	// addresses of server and peers
	var peersAdresses []string
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
	servPublicKey = moduls.RegistrationOnServer(conn)

	// Get peers' addresses
	peersNames := moduls.GetPeers(client)

	if peersNames != nil {
		for ind, name := range peersNames {
			peerAddr := moduls.PeerAddr(client, name)
			if peerAddr != nil {
				for _, ad := range peerAddr {
					peersAdresses = append(peersAdresses, ad)
					fmt.Printf("%d peer : %s  has adresse : %s \n", ind, name, ad)
				}
			}
		}
		fmt.Println("")
	} else {
		fmt.Printf("Has not peers \n")
	}

	rootPeerServ := moduls.PeerRoot(client, "jch.irif.fr")
	//keyPeerServ := moduls.PeerKey(client, "jch.irif.fr")		// doesn't return a key

	fmt.Printf("peer root : %v \n", rootPeerServ)
	fmt.Printf("peer key : %v \n", servPublicKey)

	value := moduls.GetDataByHash(conn, rootPeerServ)
	fmt.Printf("\nvalue : %v \n", value)

	//go moduls.MaintainConnectionServer(conn)

	//	reader := bufio.NewReader(os.Stdin)
	//	menu(reader, client)
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
			moduls.GetPeers(client)
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
			moduls.GetData(peer)
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
