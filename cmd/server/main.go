package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"matching-engine/internal/engine"
)

const addr = "127.0.0.1:9000"

func main() {
	e := engine.New()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Println("server listening:", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go handleConn(conn, e)
	}
}

func handleConn(conn net.Conn, e *engine.Engine) {
	defer conn.Close()

	fmt.Println("client connected:", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		e.Process(scanner.Text())
	}

	fmt.Println("client disconnected:", conn.RemoteAddr())
}