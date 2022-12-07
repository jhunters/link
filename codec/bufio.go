package codec

import (
	"bufio"
	"io"

	"github.com/jhunters/link"
)

func Bufio[S, R any](base link.Protocol[S, R], readBuf, writeBuf int) link.Protocol[S, R] {
	return &bufioProtocol[S, R]{
		base:     base,
		readBuf:  readBuf,
		writeBuf: writeBuf,
	}
}

type bufioProtocol[S, R any] struct {
	base     link.Protocol[S, R]
	readBuf  int
	writeBuf int
}

func (b *bufioProtocol[S, R]) NewCodec(rw io.ReadWriter) (cc link.Codec[S, R], err error) {
	codec := new(bufioCodec[S, R])

	if b.writeBuf > 0 {
		codec.stream.w = bufio.NewWriterSize(rw, b.writeBuf)
		codec.stream.Writer = codec.stream.w
	} else {
		codec.stream.Writer = rw
	}

	if b.readBuf > 0 {
		codec.stream.Reader = bufio.NewReaderSize(rw, b.readBuf)
	} else {
		codec.stream.Reader = rw
	}

	codec.stream.c, _ = rw.(io.Closer)

	codec.base, err = b.base.NewCodec(&codec.stream)
	if err != nil {
		return
	}
	cc = codec
	return
}

type bufioStream struct {
	io.Reader
	io.Writer
	c io.Closer
	w *bufio.Writer
}

func (s *bufioStream) Flush() error {
	if s.w != nil {
		return s.w.Flush()
	}
	return nil
}

func (s *bufioStream) close() error {
	if s.c != nil {
		return s.c.Close()
	}
	return nil
}

type bufioCodec[S, R any] struct {
	base   link.Codec[S, R]
	stream bufioStream
}

func (c *bufioCodec[S, R]) Send(msg S) error {
	if err := c.base.Send(msg); err != nil {
		return err
	}
	return c.stream.Flush()
}

func (c *bufioCodec[S, R]) Receive() (R, error) {
	return c.base.Receive()
}

func (c *bufioCodec[S, R]) Close() error {
	err1 := c.base.Close()
	err2 := c.stream.close()
	if err1 != nil {
		return err1
	}
	return err2
}
