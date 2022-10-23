// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

// DefaultOption 自主定义的解析协议
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
type Server struct{}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// ServeConn runs the server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// 解析header，选择解码器解码
// 为了实现上更简单，GeeRPC 客户端固定采用 JSON 编码 Option，后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定
// io.ReadWriteCloser 就是一个具体的连接
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	log.Printf("server: 开始处理连接..., conn:%+v", conn)
	defer func() { _ = conn.Close() }()
	var opt Option
	// 将请求中的opt数据解析出来放到opt对象中
	log.Printf("server: 开始解析opt...")
	if err := json.NewDecoder(conn).Decode(&opt); err != nil { // 客户端cc.Write的时候才会执行
		log.Println("rpc server: options error: ", err)
		return
	}
	log.Printf("server: 解析opt结束")
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	// 根据请求体编码的类型，取出对应的解码器
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	log.Printf("开始解析req")
	server.serveCodec(f(conn))
	log.Printf("处理连接结束...conn:%+v", conn)
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled
	for {                      // 之前提到过，在一次连接中，允许接收多个请求，即多个 request header 和 request body，因此这里使用了 for 无限制地等待请求的到来，直到发生错误
		// 从conn读数据到h和body
		log.Printf("server: read request")
		req, err := server.readRequest(cc)
		log.Printf("server: read request finish")
		if err != nil {
			if req == nil {
				break // it's not possible to recover, so close the connection
			}
			req.h.Error = err.Error() // err写到header里
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 拿到req，做业务处理
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

// request stores all information of a call
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	// 从conn中读取写到h中
	if err := cc.ReadHeader(&h); err != nil { // 数据是从哪来？
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: now we don't know the type of request argv
	// day 1, just suppose it's string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	log.Printf("readRequest 从连接中解析出req: %+v", req)
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	defer wg.Done()
	log.Printf("handleRequest，这里处理业务需求... %v, %v", req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("I get your req, seq: %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending) // 写入到response
	log.Printf("handleRequest finish - 已经处理完业务，写response")
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		log.Println("service: 与client建立连接")
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }
