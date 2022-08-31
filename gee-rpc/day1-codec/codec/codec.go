package codec

import (
	"io"
)

type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}

// 抽象出对消息体进行编解码
type Codec interface {
	io.Closer
	ReadHeader(*Header) error         // 反序列化
	ReadBody(interface{}) error       // 反序列化
	Write(*Header, interface{}) error // 将header和body序列化到buf
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
