package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn), // 从conn中读取
		enc:  gob.NewEncoder(buf),  // 写到buf中
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h) // 将conn中读取，再反序列化到h里
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body) // 将conn中读取，再反序列化到body里
}

// 序列化
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	// 将h写到buf中
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	// 将body写到buf中
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	return
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
