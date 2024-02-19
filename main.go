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
	// 宛先アドレス
	dest := "8.8.8.8"
	if len(os.Args) > 1 {
		dest = os.Args[1]
	}

	// リッスン用の接続を開く
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println("エラー:", err)
		return
	}
	defer c.Close()

	// ipv4.PacketConnにラップ
	p := ipv4.NewPacketConn(c)

	// TTLを設定
	if err := p.SetTTL(10); err != nil {
		fmt.Println("TTLの設定に失敗:", err)
		return
	}

	// ICMPメッセージを作成
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1, // プロセスIDとシーケンス番号を使用
			Data: []byte("HELLO"),
		},
	}

	// メッセージをバイナリ形式にエンコード
	b, err := msg.Marshal(nil)
	if err != nil {
		fmt.Println("メッセージのマーシャルに失敗:", err)
		return
	}

	// 宛先アドレスを解決
	dst, err := net.ResolveIPAddr("ip4", dest)
	if err != nil {
		fmt.Println("宛先アドレスの解決に失敗:", err)
		return
	}

	// メッセージを送信
	if _, err := p.WriteTo(b, nil, dst); err != nil {
		fmt.Println("メッセージの送信に失敗:", err)
		return
	}
	fmt.Printf("ICMP Echo Requestを %s に送信しました\n", dest)

	// 応答を待つ
	reply := make([]byte, 1500)
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, peer, err := c.ReadFrom(reply)
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
