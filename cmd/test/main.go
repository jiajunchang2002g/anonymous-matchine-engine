package main

import (
	"fmt"
	"matching-engine/internal/engine"
)

func main() {
	e := engine.New()

	trades, _ := e.Process(engine.Order{
		ID:     1,
		Symbol: "AAPL",
		Side:   engine.Sell,
		Price:  100,
		Qty:    10,
	})

	fmt.Println("trades after first order:", trades)

	trades, _ = e.Process(engine.Order{
		ID:     2,
		Symbol: "AAPL",
		Side:   engine.Buy,
		Price:  105,
		Qty:    6,
	})

	for _, t := range trades {
		fmt.Println(t)
	}
}