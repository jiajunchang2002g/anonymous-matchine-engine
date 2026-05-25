package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	addr      = flag.String("addr", "127.0.0.1:9000", "server address")
	clients   = flag.Int("clients", 100, "number of concurrent clients")
	messages  = flag.Int("messages", 1000, "messages per client")
	timeout   = flag.Duration("timeout", 5*time.Second, "per-request timeout")
	payloadSz = flag.Int("payload", 32, "extra payload size per message")
)

type result struct {
	latency time.Duration
	err     error
}

func main() {
	flag.Parse()

	totalRequests := *clients * *messages
	results := make(chan result, totalRequests)

	start := time.Now()
	var sent int64

	var wg sync.WaitGroup
	wg.Add(*clients)

	for c := 0; c < *clients; c++ {
		go func(clientID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", *addr)
			if err != nil {
				results <- result{err: fmt.Errorf("client %d dial: %w", clientID, err)}
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)

			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(clientID)))

			for i := 0; i < *messages; i++ {
				// Build a line the server can parse or just echo back.
				// Example format:
				//   BUY AAPL 100 10 xxxxxxxx
				payload := randomString(rng, *payloadSz)
				line := fmt.Sprintf("BUY AAPL %d %d %s\n", 100+rng.Intn(10), 1+rng.Intn(5), payload)

				t0 := time.Now()
				if err := conn.SetWriteDeadline(time.Now().Add(*timeout)); err != nil {
					results <- result{err: fmt.Errorf("client %d write deadline: %w", clientID, err)}
					return
				}
				if _, err := fmt.Fprint(conn, line); err != nil {
					results <- result{err: fmt.Errorf("client %d write: %w", clientID, err)}
					return
				}

				if err := conn.SetReadDeadline(time.Now().Add(*timeout)); err != nil {
					results <- result{err: fmt.Errorf("client %d read deadline: %w", clientID, err)}
					return
				}
				reply, err := reader.ReadString('\n')
				if err != nil {
					results <- result{err: fmt.Errorf("client %d read: %w", clientID, err)}
					return
				}

				_ = reply // server response is intentionally not interpreted here

				results <- result{latency: time.Since(t0)}
				atomic.AddInt64(&sent, 1)
			}
		}(c)
	}

	wg.Wait()
	close(results)

	var latencies []time.Duration
	var errs int
	for r := range results {
		if r.err != nil {
			errs++
			fmt.Println("error:", r.err)
			continue
		}
		latencies = append(latencies, r.latency)
	}

	elapsed := time.Since(start)

	if len(latencies) == 0 {
		log.Fatalf("no successful requests, errors=%d", errs)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum time.Duration
	for _, d := range latencies {
		sum += d
	}

	avg := sum / time.Duration(len(latencies))
	p50 := latencies[len(latencies)/2]
	p95 := latencies[int(float64(len(latencies))*0.95)]
	p99 := latencies[int(float64(len(latencies))*0.99)]
	if p99 > latencies[len(latencies)-1] {
		p99 = latencies[len(latencies)-1]
	}

	fmt.Println()
	fmt.Println("stress test results")
	fmt.Println("-------------------")
	fmt.Println("address:     ", *addr)
	fmt.Println("clients:     ", *clients)
	fmt.Println("messages:    ", *messages)
	fmt.Println("total reqs:  ", totalRequests)
	fmt.Println("successes:   ", len(latencies))
	fmt.Println("errors:      ", errs)
	fmt.Println("elapsed:     ", elapsed)
	fmt.Printf("throughput:  %.2f req/s\n", float64(len(latencies))/elapsed.Seconds())
	fmt.Println("avg latency: ", avg)
	fmt.Println("p50 latency: ", p50)
	fmt.Println("p95 latency: ", p95)
	fmt.Println("p99 latency: ", p99)
}

func randomString(rng *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	if n <= 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(n)

	for i := 0; i < n; i++ {
		b.WriteByte(letters[rng.Intn(len(letters))])
	}

	return b.String()
}