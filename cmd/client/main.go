package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

const addr = "127.0.0.1:9000"

func main() {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("connected")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		_, err := fmt.Fprintln(conn, scanner.Text())
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}