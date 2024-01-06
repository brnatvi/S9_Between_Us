package main

import (
	"bufio"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"client.go/moduls"
)

const TIMEOUT = 1 * time.Second

const (
	SERVER_NAME_IDX  = 1
	PEER_NAME_IDX    = 2
	MODE_IDX         = 3
	CMD_IDX          = 4
	PEER_IDX         = 5
	HASH_IDX         = 6
	REMOTE_PATH_IDX  = 6
	DOWNLOAD_DIR_IDX = 7
)

const (
	MODE_CLIENT = "Client"
	MODE_SERVER = "Server"
	MODE_MENU   = "Menu"
)

func main() {
	if len(os.Args)-1 < 3 {
		moduls.PrintError("Wrong console arguments")
		printHelp()
		return
	}

	// init params and create merkel tree
	myPeer, port, dirPath := readConfig("config")

	// Create TCP client
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   TIMEOUT,
	}

	moduls.GenerateKeys()

	if MODE_CLIENT == os.Args[MODE_IDX] {
		processClient(client)

	} else if MODE_SERVER == os.Args[MODE_IDX] {

		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", port))

		fmt.Printf("Address %v\n", addr)

		moduls.HandlePanicError(err, "[ERROR]: err resolving address ")

		//	conn, err := net.ListenUDP("udp", addr)
		//	if moduls.LOG_PRINT_DATA {
		//		fmt.Printf("Listening on port %s", port)
		//	}
		//
		serverStringAddr := GetServerAdresses(client)[0]

		fmt.Println(serverStringAddr)

		// Create UDP connection with server
		serverAddr, err := net.ResolveUDPAddr("udp", serverStringAddr)
		moduls.HandleFatalError(err, "ResolveUDPAddr failure")

		serverConn, err := net.DialUDP("udp", addr, serverAddr)
		moduls.HandleFatalError(err, "DialUDP failure")

		root := moduls.Merkelify(dirPath)
		fmt.Printf("my root: name %s, type %d, offset %d, hash %v, children %v\n",
			root.Name,
			root.NodeType,
			root.Offset,
			root.Hash,
			root.Children)

		moduls.RegistrationOnServer(serverConn, myPeer, &root)

		timePing := time.Now()

		buffer := make([]byte, moduls.DATAGRAM_SIZE)

		for {
			if time.Now().Sub(timePing) >= 10*time.Second {
				moduls.MaintainConnectionServer(serverConn, &root)
				timePing = time.Now()
			}

			serverConn.SetReadDeadline(time.Now().Add(TIMEOUT)) // set Timeout

			l, remoteAddr, err := serverConn.ReadFromUDP(buffer)

			if err != nil {
				if e, ok := err.(net.Error); !ok || !e.Timeout() {
					moduls.HandlePanicError(err, fmt.Sprintf("[ERROR] reading message from %s: ", remoteAddr))
				} else if e.Timeout() {
					//fmt.Printf("Timeout ...\n")
				}
			}
			if l > 0 {
				fmt.Printf("Receive request %d from %v\n",
					l,
					remoteAddr)
				moduls.ReplyToIncoming(serverConn, remoteAddr, buffer, moduls.Root, myPeer)
			}
		}
	} else if MODE_MENU == os.Args[MODE_IDX] {

		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", port))
		moduls.HandlePanicError(err, "[ERROR]: err resolving address ")
		conn, err := net.ListenUDP("udp", addr)
		if moduls.LOG_PRINT_DATA {
			fmt.Printf("Listening on port %s", port)
		}

		serverStringAddr := moduls.PeerAddr(client, "jch.irif.fr")
		// Create UDP connection with server
		serverAddr, err := net.ResolveUDPAddr("udp", serverStringAddr[0])
		moduls.HandleFatalError(err, "ResolveUDPAddr failure")

		serverConn, err := net.DialUDP("udp", nil, serverAddr)
		moduls.HandleFatalError(err, "DialUDP failure")

		root := moduls.Merkelify(dirPath)
		fmt.Printf("my root: name %s, type %d, offset %d, hash %v, children %v\n",
			root.Name,
			root.NodeType,
			root.Offset,
			root.Hash,
			root.Children)

		go moduls.MaintainConnectionServer(serverConn, &root)

		reader := bufio.NewReader(os.Stdin)
		go menu(reader, client)

		for {
			root = moduls.Merkelify(dirPath)

			buffer := make([]byte, moduls.DATAGRAM_SIZE)
			_, remoteAddr, err := conn.ReadFromUDP(buffer)
			moduls.HandlePanicError(err, fmt.Sprintf("[ERROR] reading message from %s: ", remoteAddr))
			moduls.ReplyToIncoming(conn, remoteAddr, buffer, root, myPeer)

		}

	}
}

func processClient(client *http.Client) {
	if len(os.Args)-1 < 4 {
		moduls.PrintError("Wrong console arguments")
		printHelp()
		return
	}

	switch os.Args[CMD_IDX] {
	case "ServerInfo":
		moduls.GetAllPeersAdresses(client)

	case "PeerInfo":
		if len(os.Args)-1 < 5 {
			moduls.PrintError("Wrong console arguments")
			printHelp()
			return
		}

		fmt.Printf("Peer {%s} info\n", os.Args[PEER_IDX])
		fmt.Printf(" addresses %v\n", moduls.PeerAddr(client, os.Args[PEER_IDX]))
		fmt.Printf(" key %s\n", hex.EncodeToString(moduls.PeerKey(client, os.Args[PEER_IDX])))
		fmt.Printf(" root %s\n", hex.EncodeToString(moduls.PeerRoot(client, os.Args[PEER_IDX])))

	case "HashesInfo", "DownloadHash", "DownloadPath":
		if len(os.Args)-1 < 5 {
			moduls.PrintError("Wrong console arguments")
			printHelp()
			return
		}

		//========= Create UDP connection with server
		addr, err := net.ResolveUDPAddr("udp", GetServerAdresses(client)[0])
		moduls.HandleFatalError(err, "ResolveUDPAddr failure")

		conn, err := net.DialUDP("udp", nil, addr)
		moduls.HandleFatalError(err, "DialUDP server failure")

		//========= Register on Server
		servPublicKey := moduls.RegistrationOnServer(conn, os.Args[PEER_NAME_IDX], nil) // empty dirpath = sharing nothing
		fmt.Printf("Connected to server { %s }\n - Public key : %v\n", os.Args[SERVER_NAME_IDX], servPublicKey)
		moduls.KeyServer = moduls.ParcePublicKay(servPublicKey)

		//========= Create UDP connection with peer
		peerAdresses := moduls.PeerAddr(client, os.Args[PEER_IDX])
		fmt.Printf("Peer's adresses %v\n", peerAdresses)

		rootPeer := moduls.PeerRoot(client, os.Args[PEER_IDX])

		moduls.KeyPeer = moduls.ParcePublicKay(moduls.PeerKey(client, os.Args[PEER_IDX]))

		addrPeer, err := net.ResolveUDPAddr("udp", peerAdresses[0])
		moduls.HandleFatalError(err, "ResolveUDPAddr failure")

		connPeer, err := net.DialUDP("udp", nil, addrPeer)
		moduls.HandleFatalError(err, "DialUDP peer failure")

		if moduls.NatTraversal(client, conn, connPeer, os.Args[PEER_NAME_IDX], os.Args[PEER_IDX]) == moduls.RESULT_OK {
			fmt.Printf("\nNatTraversal OK  --> Connected to peer { %s }\n", os.Args[PEER_IDX])
		} else {
			fmt.Printf("\nNatTraversal NotOK --> Not connected to peer { %s }\n", os.Args[PEER_IDX])
		}

		if "HashesInfo" == os.Args[CMD_IDX] {
			DataObj := moduls.DataObject{moduls.OP_PRINT_HASH, moduls.NODE_UNKNOWN, "", "/", "", ".", nil}
			moduls.DownloadData(connPeer, rootPeer, os.Args[PEER_NAME_IDX], &DataObj)

		} else {
			if len(os.Args)-1 < 7 {
				moduls.PrintError("Wrong console arguments")
				printHelp()
				return
			}

			outputDir := os.Args[DOWNLOAD_DIR_IDX]
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				moduls.PrintError("Output directory isn't existis!")
				return
			}

			if "DownloadHash" == os.Args[CMD_IDX] {
				hash, err := hex.DecodeString(os.Args[HASH_IDX])
				if err != nil || len(hash) != 32 {
					moduls.PrintError("Decoding hash error")
					return
				}
				DataObj := moduls.DataObject{moduls.OP_DOWNLOAD_HASH, moduls.NODE_UNKNOWN, "", "", "", outputDir, nil}
				moduls.DownloadData(connPeer, hash, os.Args[PEER_NAME_IDX], &DataObj)
			} else { //Download path
				DataObj := moduls.DataObject{moduls.OP_DOWNLOAD_PATH, moduls.NODE_UNKNOWN, "", "/", os.Args[REMOTE_PATH_IDX], outputDir, nil}
				moduls.DownloadData(connPeer, rootPeer, os.Args[PEER_NAME_IDX], &DataObj)
			}
		}

		conn.Close()

	default:
		moduls.PrintError("Wrong console arguments")
		printHelp()
		return
	}
}

func printHelp() {
	fmt.Print("usage:\n")
	fmt.Print("go client.go ServerName MyPeerName Mode [... extra parameters]:\n")
	fmt.Print("  Mode: can have 3 values: Client, Server, Menu\n")
	fmt.Print("For **Client** mode next operations are avalable:\n")
	fmt.Print("  ServerInfo - display on the screen list of the peers, address, keys, root\n")
	fmt.Print("  PeerInfo - display on the screen list of the peers, address, keys, root\n")
	fmt.Print("  HashesInfo - display on the screen hashes and associated names\n")
	fmt.Print("  DownloadHash - download data by hash\n")
	fmt.Print("   Example: go client.go ServerName MyPeerName Client DownloadHash Peer HASH DownloadDir\n")
	fmt.Print("            Where Peer is a peer name\n")
	fmt.Print("            Where HASH is 64 char string composed of hex literals\n")
	fmt.Print("            Where DownloadDir is output directory on local HDD\n")
	fmt.Print("  DownloadPath - download data by path\n")
	fmt.Print("   Example: go client.go ServerName MyPeerName Client DownloadPath Peer PATH DownloadDir\n")
	fmt.Print("            Where Peer is a peer name\n")
	fmt.Print("            Where PATH is path on remote peer, for example /images/teachers.jpg\n")
	fmt.Print("            Where DownloadDir is output directory on local HDD\n")
	fmt.Print("For **Server** mode next operations are avalable:\n")
	fmt.Print("  TODO\n")
	fmt.Print("For **Menu** there is no extra parameters\n")
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

	// TODO p -d interactions(?) after first request
	for {
		fmt.Print(`
prot serv int!! ^_^
Options:
	list: lists existing peers
	p -a: shows p's addresses
	p -k: shows p's public key (if it has one)
	p -r: shows p's root hash
	p -d: prompt to ask for hash to request from peer p
	files: (on hold)
	exit: exits
=>`)
		cmd, err := reader.ReadString('\n')
		cmd = cmd[:len(cmd)-1]
		moduls.HandlePanicError(err, "[ERROR] Read err!")

		command, peer := parseCmd(cmd)
		switch command {
		case 0:
			moduls.GetAllPeersAdresses(client)
		case 1:
			addrs := moduls.PeerAddr(client, peer)
			fmt.Printf("%s 's addresses : \n", peer)
			fmt.Println(addrs)
		case 2:
			key := moduls.PeerKey(client, peer)
			fmt.Printf("%s 's key : \n", peer)
			fmt.Println(key)
		case 3:
			root := moduls.PeerRoot(client, peer)
			fmt.Printf("%s 's root hash : \n", peer)
			fmt.Printf("%x \n", root)
		case 4:
			fmt.Printf("Requested hash:")
			hash, err := reader.ReadString('\n')
			moduls.HandlePanicError(err, "[ERROR] read err ")
			moduls.GetData(client, peer, hash)
			reader.Discard(reader.Buffered())
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
