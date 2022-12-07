package codec

import (
	"bytes"
	"testing"

	"github.com/jhunters/link"

	. "github.com/smartystreets/goconvey/convey"
)

type MyMessage1 struct {
	Field1 string
	Field2 int
}

type MyMessage2 struct {
	Field1 int
	Field2 string
}

func JsonTestProtocol() *JsonProtocol[MyMessage1, MyMessage1] {
	protocol := Json[MyMessage1, MyMessage1]()
	protocol.Register(MyMessage1{})
	protocol.RegisterName("msg2", &MyMessage2{})
	return protocol
}

func codecTest(t *testing.T, protocol link.Protocol[MyMessage1, MyMessage1]) {

	Convey("codecTest", t, func() {
		var stream bytes.Buffer

		codec, _ := protocol.NewCodec(&stream)

		sendMsg1 := MyMessage1{
			Field1: "abc",
			Field2: 123,
		}

		err := codec.Send(sendMsg1)
		if err != nil {
			t.Fatal(err)
		}

		recvMsg1, err := codec.Receive()
		if err != nil {
			t.Fatal(err)
		}

		So(sendMsg1, ShouldResemble, recvMsg1)
	})

}

func Test_Json(t *testing.T) {
	protocol := JsonTestProtocol()
	codecTest(t, protocol)
}
