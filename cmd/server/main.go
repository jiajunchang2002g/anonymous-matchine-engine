package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"errors"

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
		line := scanner.Text()

		order, err := parseOrder(line)
		if err != nil {
			fmt.Fprintln(conn, "ERR", err)
			continue
		}

		trades, err := e.Process(order)
		if err != nil {
			fmt.Fprintln(conn, "ERR", err)
			continue
		}

		for _, t := range trades {
			fmt.Fprintln(conn, t)
		}
	}

	fmt.Println("client disconnected:", conn.RemoteAddr())
}

func parseOrder(line string) (engine.Order, error) {
	fields := strings.Fields(line)

	if len(fields) != 4 {
		return engine.Order{}, errors.New("invalid format")
	}

	sideStr := fields[0]
	symbol := fields[1]

	price, err := strconv.Atoi(fields[2])
	if err != nil {
		return engine.Order{}, err
	}

	qty, err := strconv.Atoi(fields[3])
	if err != nil {
		return engine.Order{}, err
	}

	var side engine.Side

	switch sideStr {
	case "BUY":
		side = engine.Buy

	case "SELL":
		side = engine.Sell

	default:
		return engine.Order{}, errors.New("invalid side")
	}

	return engine.Order{
		ID:     time.Now().UnixNano(),
		Symbol: symbol,
		Side:   side,
		Price:  price,
		Qty:    qty,
	}, nil
}