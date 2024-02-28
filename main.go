package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type HopResult struct {
	Hop       int
	Host      string
	ResultStr string
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
	err = (*c).SetReadDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return "", nil, fmt.Errorf("failed to read reply: %v", err)
	}

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
		return "", rm.Type, fmt.Errorf("unexpected message type: %v", rm.Type)
	}
}

func main() {
	startTime := time.Now()

	// deamonで動作しているときは，死んだ瞬間をテキストファイルに記述する．
	deamon := flag.Bool("d", false, "Enable deamon mode")
	flag.Parse()

	logFilePath := fmt.Sprintf("logs/%s.log", startTime.Format("20060102_150405"))

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	// ログの出力先を設定
	log.SetOutput(logFile)

	// ログレベルの設定（オプション）
	log.SetFlags(log.Ldate | log.Ltime)

	// フラグ以外の引数を取得（宛先アドレスなど）
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: <program> [-d] destination")
		os.Exit(1)
	}
	dest := args[0] // 宛先アドレス

	// Open a connection for listening
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer c.Close()

	maxHop := 64
	var hops []HopResult

	fmt.Println("Tracing route to ", dest)
	for i := 1; i < maxHop; i++ {
		replyHost, msgType, err := sendICMPEchoRequest(dest, i, &c)
		if err != nil {
			fmt.Printf("Error: %v at %d \n", err, i)
			//hops = append(hops, HopResult{Hop: i, Host: ""})
		}
		hops = append(hops, HopResult{Hop: i, Host: replyHost})
		if msgType == ipv4.ICMPTypeEchoReply {
			break
		}
	}

	//results := make(map[string]string)
	//resultChan := make(chan map[string]string)
	resultChan := make(chan HopResult)
	var wg sync.WaitGroup

	for _, hop := range hops {
		wg.Add(1)
		go func(hop HopResult) {
			for {
				if hop.Host != "" {
					_, icmpType, err := sendICMPEchoRequest(hop.Host, 64, &c)
					result := "Error"
					if err == nil {
						result = fmt.Sprintf("%v", icmpType)
					}
					resultChan <- HopResult{Hop: hop.Hop, Host: hop.Host, ResultStr: result}

					time.Sleep(1 * time.Second)
				}
			}
		}(hop)
	}

	go func() {
		for result := range resultChan {
			// 結果を更新
			for i, hop := range hops {
				if hop.Hop == result.Hop {
					hops[i].ResultStr = result.ResultStr
					break
				}
			}
			// 結果をホップ番号でソート
			sort.Slice(hops, func(i, j int) bool {
				return hops[i].Hop < hops[j].Hop
			})

			if *deamon {
				for _, hop := range hops {
					if hop.ResultStr != "echo reply" {
						log.Printf("%d: %s : %s\n", hop.Hop, hop.Host, hop.ResultStr)
					}
				}
			} else {
				for _, hop := range hops {
					fmt.Printf("%d: %s : %s\n", hop.Hop, hop.Host, hop.ResultStr)
				}
				fmt.Printf("--------------------------------------------------\n")
			}
		}
	}()

	wg.Wait()

}
