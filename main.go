package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	dest := "8.8.8.8"
	if len(os.Args) > 1 {
		dest = os.Args[1]
	}

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("HELLO"),
		},
	}

	bin, err := msg.Marshal(nil)
	if err != nil {
		fmt.Println("failed at encoding message: ", err)
		return
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println("failed at opening connection", err)
		return
	}
	defer conn.Close()

	// 宛先のIPアドレスを解決
	addr, err := net.ResolveIPAddr("ip4", dest)
	if err != nil {
		fmt.Println("IPアドレスの解決に失敗しました:", err)
		return
	}

	// ICMP Echo Requestを送信
	if _, err := conn.WriteTo(bin, addr); err != nil {
		fmt.Println("メッセージの送信に失敗しました:", err)
		return
	}
	fmt.Printf("ICMP Echo Requestを %s に送信しました\n", addr)

	// 応答を待つ
	reply := make([]byte, 1500)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, peer, err := conn.ReadFrom(reply)
	if err != nil {
		fmt.Println("応答の受信に失敗しました:", err)
		return
	}

	// 応答メッセージを解析
	rm, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		fmt.Println("応答メッセージの解析に失敗しました:", err)
		return
	}

	if rm.Type == ipv4.ICMPTypeEchoReply {
		fmt.Printf("ICMP Echo Replyを %s から受信しました\n", peer)
	} else {
		fmt.Println("期待していないメッセージタイプ:", rm.Type)
	}

}
