package main

import (
	"log"

	"github.com/jhunters/link"
	"github.com/jhunters/link/codec"
)

type AddReq struct {
	A, B int
}

type AddRsp struct {
	C int
}

func main() {
	serverJson := codec.Json[AddRsp, AddReq]()
	serverJson.Register(AddReq{})
	serverJson.Register(AddRsp{})

	server, err := link.Listen[AddRsp, AddReq]("tcp", "0.0.0.0:0",
		serverJson, 0 /* sync send */, link.HandlerFunc[AddRsp, AddReq](serverSessionLoop))
	checkErr(err)
	addr := server.Listener().Addr().String()
	go server.Serve()

	json := codec.Json[AddReq, AddRsp]()
	json.Register(AddReq{})
	json.Register(AddRsp{})

	client, err := link.Dial[AddReq, AddRsp]("tcp", addr, json, 0)
	checkErr(err)
	clientSessionLoop(client)
}

func serverSessionLoop(session *link.Session[AddRsp, AddReq]) {
	for {
		req, err := session.Receive()
		checkErr(err)

		err = session.Send(AddRsp{
			req.A + req.B,
		})
		checkErr(err)
	}
}

func clientSessionLoop(session *link.Session[AddReq, AddRsp]) {
	for i := 0; i < 10; i++ {
		err := session.Send(AddReq{
			i, i,
		})
		checkErr(err)
		log.Printf("Send: %d + %d", i, i)

		rsp, err := session.Receive()
		checkErr(err)
		log.Printf("Receive: %d", rsp.C)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
