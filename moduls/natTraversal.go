package moduls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func NatTraversalServer(message []byte, i int) *net.UDPConn {
	len := binary.BigEndian.Uint16(message[POS_LENGTH:(POS_LENGTH + LENGTH_SIZE)])
	stAddrPeer := string(message[POS_VALUE:(POS_VALUE + len)])
	fmt.Printf("Address of peer : %s\n", stAddrPeer)

	//Natalia: why port is hardcoded???
	myAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", (8443+i)))
	HandleFatalError(err, "ResolveUDPAddr failure")

	peerAddr, err := net.ResolveUDPAddr("udp", stAddrPeer)
	HandleFatalError(err, "ResolveUDPAddr failure")

	conn, err := net.DialUDP("udp", myAddr, peerAddr)
	HandleFatalError(err, "DialUDP failure")

	return conn
}

// NAT bypass function:
// first makes a maximum of 3 attempts to contact the <otherPeer>,
// if attempts are unsuccessful, sends a request to the server and waits for a "Hello" message from the <otherPeer>,
// and then sends a response message to confirm the connection
// Parameters:
// - tcpClient - TCP client with server
// - conn - UDP connection with server
// - connPeer - UDP connection with other peer
// - myPeer - name of my peer
// - otherPeer - name of other peerNatTraversalNatTraversal
// Return: RESULT_ERROR or RESULT_OK

func NatTraversal(tcpClient *http.Client, conn *net.UDPConn, connPeer *net.UDPConn, myPeer string, otherPeer string) int {
	// max 3 attempts to send Hello to otherPeer
	maxNbAtts := 1
	count := 1
	bufRes := make([]byte, DATAGRAM_SIZE)

	for count <= maxNbAtts {
		_, err := sendHello(connPeer, myPeer)
		messCounter++
		count++

		if err != nil {
			time.Sleep(1 * TIMEOUT)
			continue
		} else {
			break
		}
	}

	fmt.Printf("%d attempts to send Hello to peer { %s } was made\n", count-1, otherPeer)

	if count-1 == maxNbAtts {
		fmt.Printf("Need to proceed NatTraversal\n")

		addresses := PeerAddr(tcpClient, otherPeer)
		fmt.Println(addresses)

		ip := net.ParseIP(addresses[0])
		addrSize := isIPv4(addresses[0])
		if addrSize == -1 {
			PrintError("NatTraversal: address neither IPv4, nor IPv6")
			return RESULT_ERROR
		}

		buf := make([]byte, DATAGRAM_SIZE)

		// send NatTraversalRequest to Server
		buf = composeNatTravMessage(messCounter, byte(NAT_TRAVERSAL_REQUEST), ip, addrSize, 0)

		_, err := conn.Write(buf)
		messCounter++
		if err != nil {
			HandleFatalError(err, "NatTraversal: Write to UDP")
			return RESULT_ERROR
		}

		timeStart := time.Now()

		// max 3 attempts to recieve Hello from otherPeer
		count = 1
		bExit := false
		for count <= maxNbAtts {

			connPeer.SetReadDeadline(time.Now().Add(TIMEOUT)) // set a timeout
			count++

			// wait Hello from otherPeer
			all, _, err := connPeer.ReadFromUDP(bufRes)
			if err != nil {
				if err != io.EOF {
					HandleFatalError(err, "NatTraversal: ReadFromUDP")
					continue
				}
			}
			rezCheck := CheckUDPIncomingPacket(bufRes, all, HELLO, "HELLO")
			switch rezCheck {
			case 0:
				bExit = true
			case 1: // exit from function
				messCounter++
				PrintError("NatTraversal: The lenght of HELLO recieved != expected one\n")
				return RESULT_ERROR

			case 2: // reject and try to pull out the next response until TIMEOUT
				timeNow := time.Now()

				// if TIMEOUT -> exit from function to start all over again
				if timeStart.Sub(timeNow) >= TIMEOUT {
					messCounter++
					PrintError("NatTraversal: Timeout reception of HELLO")
					return RESULT_ERROR
				} else {
					continue
				}
			}

			if bExit {
				break
			}
		}
		messCounter++
		fmt.Printf("%d attempts to recieve Hello from peer { %s } was made\n", count-1, otherPeer)

		if bExit {
			// send HelloReply to otherPeer
			buf = composeHandChakeMessage(messCounter, byte(HELLO_REPLY), myPeer, 0, 0)
			_, err = connPeer.Write(buf)
			messCounter++
			if err != nil {
				HandleFatalError(err, "NatTraversal: Write to UDP HelloReply failure")
				return RESULT_ERROR
			}

			// send Hello to otherPeer
			buf = composeHandChakeMessage(messCounter, byte(HELLO), myPeer, 0, 0)
			_, err = connPeer.Write(buf)
			messCounter++
			if err != nil {
				HandleFatalError(err, "NatTraversal: Write to UDP Hello failure")
				return RESULT_ERROR
			}
			messCounter++
			return RESULT_OK
		} else {
			return RESULT_ERROR
		}
	} else {
		return RESULT_OK
	}
}

// Composes UDP message to send NatTravessal request and converts it to binary
func composeNatTravMessage(idMes uint32, typeMes uint8, addr []byte, lenMes int, extentMes int) []byte {

	var buf bytes.Buffer

	i := make([]byte, 4)
	binary.BigEndian.PutUint32(i, idMes)
	buf.Write(i)

	buf.WriteByte(typeMes)

	j := make([]byte, 2)
	binary.BigEndian.PutUint16(j, uint16(lenMes))
	buf.Write(j)

	buf.Write(addr)

	k := make([]byte, 4)
	binary.BigEndian.PutUint32(k, uint32(extentMes))
	buf.Write(k)

	return buf.Bytes()
}

// Cheque if IP adress is is IPv4 or IPv6
// Parameter: string address
// Return: size in byte of address
//  6 for IPv4
// 18 for IPv6
// -1 else
func isIPv4(addr string) int {
	a := strings.Split(addr, ":")
	ip := net.ParseIP(a[0])
	if ip == nil {
		PrintError("IP address is not valid\n")
		return -1
	}

	if ip.To4() != nil {
		return 6
	}
	return 18
}
