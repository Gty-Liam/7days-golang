package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("server: start rpc server on", l.Addr())
	addr <- l.Addr().String() // 等待连接
	geerpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	time.Sleep(time.Second * 1)

	// in fact, following code is like a simple geerpc client
	log.Printf("clent: 尝试建立连接")
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	log.Printf("sleep...\n")
	time.Sleep(time.Second)

	// send options
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response

	for i := 1; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		// head, body
		log.Printf("写请求： header: %+v | body: %+v \n", h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq)) // 把h和body写到buf中
		time.Sleep(time.Second * 1)

		var header = &codec.Header{}
		time.Sleep(1000)
		log.Printf("处理完读resp ↓")
		_ = cc.ReadHeader(header) // 从conn中读取到header
		var reply string
		_ = cc.ReadBody(&reply) // 从conn中读取body
		log.Println("header:", header)
		log.Println("reply:", reply)
		log.Println("/***************/")
		time.Sleep(1000)
	}

}
