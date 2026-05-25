package engine

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type Order struct {
	ID       int64
	ClientID string
	Symbol   string
	Side     Side
	Price    int
	Qty      int
}

type Trade struct {
	BuyOrderID  int64
	SellOrderID int64
	Symbol      string
	Price       int
	Qty         int
}

type Engine struct {
	mu    sync.Mutex
	books map[string]*book
}

type book struct {
	bids *sideBook
	asks *sideBook
}

type sideBook struct {
	prices []int
	levels map[int][]Order
}

func New() *Engine {
	return &Engine{
		books: make(map[string]*book),
	}
}

func (e *Engine) Process(o Order) ([]Trade, error) {
	if o.ID == 0 {
		return nil, errors.New("order id must be non-zero")
	}
	if o.Symbol == "" {
		return nil, errors.New("symbol required")
	}
	if o.Side != Buy && o.Side != Sell {
		return nil, errors.New("side must be BUY or SELL")
	}
	if o.Price <= 0 {
		return nil, errors.New("price must be positive")
	}
	if o.Qty <= 0 {
		return nil, errors.New("qty must be positive")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	b := e.getBook(o.Symbol)
	return b.process(o), nil
}

func (e *Engine) getBook(symbol string) *book {
	b, ok := e.books[symbol]
	if !ok {
		b = &book{
			bids: newSideBook(),
			asks: newSideBook(),
		}
		e.books[symbol] = b
	}
	return b
}

func newSideBook() *sideBook {
	return &sideBook{
		levels: make(map[int][]Order),
	}
}

func (b *book) process(o Order) []Trade {
	var trades []Trade

	switch o.Side {
	case Buy:
		for o.Qty > 0 {
			bestAsk, ok := b.asks.bestAsk()
			if !ok || bestAsk > o.Price {
				break
			}

			queue := b.asks.levels[bestAsk]
			resting := queue[0]
			tradedQty := min(o.Qty, resting.Qty)

			trades = append(trades, Trade{
				BuyOrderID:  o.ID,
				SellOrderID: resting.ID,
				Symbol:      o.Symbol,
				Price:       bestAsk,
				Qty:         tradedQty,
			})

			o.Qty -= tradedQty
			resting.Qty -= tradedQty

			if resting.Qty == 0 {
				queue = queue[1:]
			} else {
				queue[0] = resting
			}

			if len(queue) == 0 {
				b.asks.removeLevel(bestAsk)
			} else {
				b.asks.levels[bestAsk] = queue
			}
		}

		if o.Qty > 0 {
			b.bids.add(o)
		}

	case Sell:
		for o.Qty > 0 {
			bestBid, ok := b.bids.bestBid()
			if !ok || bestBid < o.Price {
				break
			}

			queue := b.bids.levels[bestBid]
			resting := queue[0]
			tradedQty := min(o.Qty, resting.Qty)

			trades = append(trades, Trade{
				BuyOrderID:  resting.ID,
				SellOrderID: o.ID,
				Symbol:      o.Symbol,
				Price:       bestBid,
				Qty:         tradedQty,
			})

			o.Qty -= tradedQty
			resting.Qty -= tradedQty

			if resting.Qty == 0 {
				queue = queue[1:]
			} else {
				queue[0] = resting
			}

			if len(queue) == 0 {
				b.bids.removeLevel(bestBid)
			} else {
				b.bids.levels[bestBid] = queue
			}
		}

		if o.Qty > 0 {
			b.asks.add(o)
		}
	}

	return trades
}

func (s *sideBook) add(o Order) {
	if _, ok := s.levels[o.Price]; !ok {
		s.insertPrice(o.Price)
	}
	s.levels[o.Price] = append(s.levels[o.Price], o)
}

func (s *sideBook) insertPrice(p int) {
	i := sort.SearchInts(s.prices, p)
	s.prices = append(s.prices, 0)
	copy(s.prices[i+1:], s.prices[i:])
	s.prices[i] = p
}

func (s *sideBook) removeLevel(p int) {
	delete(s.levels, p)
	i := sort.SearchInts(s.prices, p)
	if i < len(s.prices) && s.prices[i] == p {
		s.prices = append(s.prices[:i], s.prices[i+1:]...)
	}
}

func (s *sideBook) bestAsk() (int, bool) {
	if len(s.prices) == 0 {
		return 0, false
	}
	return s.prices[0], true
}

func (s *sideBook) bestBid() (int, bool) {
	if len(s.prices) == 0 {
		return 0, false
	}
	return s.prices[len(s.prices)-1], true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t Trade) String() string {
	return fmt.Sprintf(
		"TRADE symbol=%s price=%d qty=%d buy=%d sell=%d",
		t.Symbol, t.Price, t.Qty, t.BuyOrderID, t.SellOrderID,
	)
}