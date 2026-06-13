package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

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
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch strings.ToUpper(fields[0]) {
		case "CANCEL":
			if len(fields) < 2 {
				fmt.Fprintln(conn, "ERR cancel requires an order id")
				continue
			}
			id, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				fmt.Fprintln(conn, "ERR invalid order id:", err)
				continue
			}
			if err := e.Cancel(id); err != nil {
				fmt.Fprintln(conn, "ERR", err)
				continue
			}
			fmt.Fprintln(conn, "CANCELLED", id)

		default:
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

			if len(trades) == 0 {
				fmt.Fprintln(conn, "ACK", order.ID)
			} else {
				for _, t := range trades {
					fmt.Fprintln(conn, t)
				}
			}
		}
	}

	fmt.Println("client disconnected:", conn.RemoteAddr())
}

func parseOrder(line string) (engine.Order, error) {
	fields := strings.Fields(line)

	// Accept 4 or more fields; extra trailing fields are ignored.
	if len(fields) < 4 {
		return engine.Order{}, errors.New("invalid format: expected <SIDE> <SYMBOL> <PRICE> <QTY>")
	}

	sideStr := fields[0]
	symbol := fields[1]

	price, err := strconv.Atoi(fields[2])
	if err != nil {
		return engine.Order{}, fmt.Errorf("invalid price: %w", err)
	}

	qty, err := strconv.Atoi(fields[3])
	if err != nil {
		return engine.Order{}, fmt.Errorf("invalid qty: %w", err)
	}

	var side engine.Side

	switch strings.ToUpper(sideStr) {
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