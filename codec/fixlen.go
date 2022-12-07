package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/jhunters/link"
)

var ErrTooLargePacket = errors.New("Too Large Packet")

type FixLenProtocol[S, R any] struct {
	base        link.Protocol[S, R]
	n           int
	maxRecv     int
	maxSend     int
	headDecoder func([]byte) int
	headEncoder func([]byte, int)
}

func FixLen[S, R any](base link.Protocol[S, R], n int, byteOrder binary.ByteOrder, maxRecv, maxSend int) *FixLenProtocol[S, R] {
	proto := &FixLenProtocol[S, R]{
		n:    n,
		base: base,
	}
	switch n {
	case 1:
		if maxRecv > math.MaxUint8 {
			maxRecv = math.MaxUint8
		}
		if maxSend > math.MaxUint8 {
			maxSend = math.MaxUint8
		}
		proto.headDecoder = func(b []byte) int {
			return int(b[0])
		}
		proto.headEncoder = func(b []byte, size int) {
			b[0] = byte(size)
		}
	case 2:
		if maxRecv > math.MaxUint16 {
			maxRecv = math.MaxUint16
		}
		if maxSend > math.MaxUint16 {
			maxSend = math.MaxUint16
		}
		proto.headDecoder = func(b []byte) int {
			return int(byteOrder.Uint16(b))
		}
		proto.headEncoder = func(b []byte, size int) {
			byteOrder.PutUint16(b, uint16(size))
		}
	case 4:
		if maxRecv > math.MaxUint32 {
			maxRecv = math.MaxUint32
		}
		if maxSend > math.MaxUint32 {
			maxSend = math.MaxUint32
		}
		proto.headDecoder = func(b []byte) int {
			return int(byteOrder.Uint32(b))
		}
		proto.headEncoder = func(b []byte, size int) {
			byteOrder.PutUint32(b, uint32(size))
		}
	case 8:
		proto.headDecoder = func(b []byte) int {
			return int(byteOrder.Uint64(b))
		}
		proto.headEncoder = func(b []byte, size int) {
			byteOrder.PutUint64(b, uint64(size))
		}
	default:
		panic("FixLenProtocol: unsupported head size")
	}
	proto.maxRecv = maxRecv
	proto.maxSend = maxSend
	return proto
}

func (p *FixLenProtocol[S, R]) NewCodec(rw io.ReadWriter) (cc link.Codec[S, R], err error) {
	codec := &fixlenCodec[S, R]{
		rw:             rw,
		FixLenProtocol: p,
	}
	codec.headBuf = codec.head[:p.n]

	codec.base, err = p.base.NewCodec(&codec.fixlenReadWriter)
	if err != nil {
		return
	}
	cc = codec
	return
}

type fixlenReadWriter struct {
	recvBuf bytes.Reader
	sendBuf bytes.Buffer
}

func (rw *fixlenReadWriter) Read(p []byte) (int, error) {
	return rw.recvBuf.Read(p)
}

func (rw *fixlenReadWriter) Write(p []byte) (int, error) {
	return rw.sendBuf.Write(p)
}

type fixlenCodec[S, R any] struct {
	base    link.Codec[S, R]
	head    [8]byte
	headBuf []byte
	bodyBuf []byte
	rw      io.ReadWriter
	*FixLenProtocol[S, R]
	fixlenReadWriter
}

func (c *fixlenCodec[S, R]) Receive() (R, error) {
	var body R
	if _, err := io.ReadFull(c.rw, c.headBuf); err != nil {
		return body, err
	}
	size := c.headDecoder(c.headBuf)
	if size > c.maxRecv {
		return body, ErrTooLargePacket
	}
	if cap(c.bodyBuf) < size {
		c.bodyBuf = make([]byte, size, size+128)
	}
	buff := c.bodyBuf[:size]
	if _, err := io.ReadFull(c.rw, buff); err != nil {
		return body, err
	}
	c.recvBuf.Reset(buff)
	msg, err := c.base.Receive()
	return msg, err
}

func (c *fixlenCodec[S, R]) Send(msg S) error {
	c.sendBuf.Reset()
	c.sendBuf.Write(c.headBuf)
	err := c.base.Send(msg)
	if err != nil {
		return err
	}
	buff := c.sendBuf.Bytes()
	c.headEncoder(buff, len(buff)-c.n)
	_, err = c.rw.Write(buff)
	return err
}

func (c *fixlenCodec[S, R]) Close() error {
	if closer, ok := c.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
