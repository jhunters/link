package codec

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/jhunters/link"
)

type JsonProtocol[S, R any] struct {
	types map[string]reflect.Type
	names map[reflect.Type]string
}

func Json[S, R any]() *JsonProtocol[S, R] {
	return &JsonProtocol[S, R]{
		types: make(map[string]reflect.Type),
		names: make(map[reflect.Type]string),
	}
}

func (j *JsonProtocol[S, R]) Register(t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := rt.PkgPath() + "/" + rt.Name()
	j.types[name] = rt
	j.names[rt] = name
}

func (j *JsonProtocol[S, R]) RegisterName(name string, t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	j.types[name] = rt
	j.names[rt] = name
}

func (j *JsonProtocol[S, R]) NewCodec(rw io.ReadWriter) (link.Codec[S, R], error) {
	codec := &jsonCodec[S, R]{
		p:       j,
		encoder: json.NewEncoder(rw),
		decoder: json.NewDecoder(rw),
	}
	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

type jsonIn struct {
	Head string
	Body *json.RawMessage
}

type jsonOut struct {
	Head string
	Body interface{}
}

type jsonCodec[S, R any] struct {
	p       *JsonProtocol[S, R]
	closer  io.Closer
	encoder *json.Encoder
	decoder *json.Decoder
}

func (c *jsonCodec[S, R]) Receive() (R, error) {
	var in jsonIn
	err := c.decoder.Decode(&in)
	var body R
	if err != nil {
		return body, err
	}
	if in.Head != "" {
		if t, exists := c.p.types[in.Head]; exists {
			v := reflect.New(t).Interface().(*R)
			body = *v
		}
	}
	err = json.Unmarshal(*in.Body, &body)
	if err != nil {
		return body, err
	}
	return body, nil
}

func (c *jsonCodec[S, R]) Send(msg S) error {
	var out jsonOut
	t := reflect.TypeOf(msg)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if name, exists := c.p.names[t]; exists {
		out.Head = name
	}
	out.Body = msg
	return c.encoder.Encode(&out)
}

func (c *jsonCodec[S, R]) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
