package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type HopResult struct {
	Address string
	Result  string
}

// sendICMPEchoRequest sends an ICMP Echo Request with specified destination and TTL,
// and returns the IP address of the host that sent the ICMP Echo Reply.
func sendICMPEchoRequest(destination string, TTL int, c *net.PacketConn) (string, icmp.Type, error) {
	p := ipv4.NewPacketConn(*c)

	// Set TTL
	if err := p.SetTTL(TTL); err != nil {
		return "", nil, fmt.Errorf("failed to set TTL: %v", err)
	}

	// Create an ICMP Echo Request message
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1, // Use process ID and sequence number 1
			Data: []byte("HELLO"),
		},
	}

	// Marshal the message into binary format
	bin, err := msg.Marshal(nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal message: %v", err)
	}

	// Resolve destination IP address
	dst, err := net.ResolveIPAddr("ip4", destination)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve destination address: %v", err)
	}

	// Send the ICMP Echo Request message
	if _, err := p.WriteTo(bin, nil, dst); err != nil {
		return "", nil, fmt.Errorf("failed to send message: %v", err)
	}

	// Prepare to read the reply
	reply := make([]byte, 1500)
	(*c).SetReadDeadline(time.Now().Add(3 * time.Second))

	n, peer, err := (*c).ReadFrom(reply)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read reply: %v", err)
	}

	// Parse the reply message
	rm, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse reply message: %v", err)
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		// Successfully received an ICMP Echo Reply
		peerAddr, ok := peer.(*net.IPAddr)
		if !ok {
			return "", nil, fmt.Errorf("invalid peer address type")
		}
		return peerAddr.String(), rm.Type, nil
	case ipv4.ICMPTypeTimeExceeded:
		peerAddr, ok := peer.(*net.IPAddr)
		if !ok {
			return "", nil, fmt.Errorf("invalid peer address type")
		}
		return peerAddr.String(), rm.Type, nil
	default:
		return "", nil, fmt.Errorf("unexpected message type: %v", rm.Type)
	}
}

func main() {
	dest := "202.224.52.158"
	//dest := "192.168.150.1"
	if len(os.Args) > 1 {
		dest = os.Args[1]
	}

	// Open a connection for listening
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer c.Close()

	maxHop := 64
	var hops []string

	for i := 1; i < maxHop; i++ {
		replyHost, msgType, err := sendICMPEchoRequest(dest, i, &c)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		hops = append(hops, replyHost)
		if msgType == ipv4.ICMPTypeEchoReply {
			break
		}
	}

	fmt.Println(hops)

	results := make(map[string]string)
	resultChan := make(chan map[string]string)
	//resultChan := make(chan HopResult, len(hops))
	for _, hop := range hops {
		go func(hop string) {
			for {
				_, icmpType, err := sendICMPEchoRequest(hop, 64, &c)
				if err != nil {
					results[hop] = fmt.Sprintf("Error: %v", err)
				} else {
					results[hop] = fmt.Sprintf("Recv %v", icmpType)
				}
				resultChan <- results

				time.Sleep(1 * time.Second)
			}
		}(hop)
	}

	for updatedResults := range resultChan {
		for hop, result := range updatedResults {
			fmt.Printf("%s : %s\n", hop, result)
		}
		fmt.Println("------")
	}
}
