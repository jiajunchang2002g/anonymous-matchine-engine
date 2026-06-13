package engine

import (
	"testing"
)

// helpers

func mustProcess(t *testing.T, e *Engine, o Order) []Trade {
	t.Helper()
	trades, err := e.Process(o)
	if err != nil {
		t.Fatalf("Process(%+v) unexpected error: %v", o, err)
	}
	return trades
}

func requireTrades(t *testing.T, got []Trade, want int) {
	t.Helper()
	if len(got) != want {
		t.Fatalf("expected %d trade(s), got %d: %+v", want, len(got), got)
	}
}

// ─── validation ────────────────────────────────────────────────────────────

func TestProcess_Validation(t *testing.T) {
	e := New()

	cases := []struct {
		name  string
		order Order
	}{
		{"zero id", Order{ID: 0, Symbol: "X", Side: Buy, Price: 10, Qty: 1}},
		{"empty symbol", Order{ID: 1, Symbol: "", Side: Buy, Price: 10, Qty: 1}},
		{"bad side", Order{ID: 1, Symbol: "X", Side: "HOLD", Price: 10, Qty: 1}},
		{"zero price", Order{ID: 1, Symbol: "X", Side: Buy, Price: 0, Qty: 1}},
		{"negative price", Order{ID: 1, Symbol: "X", Side: Buy, Price: -1, Qty: 1}},
		{"zero qty", Order{ID: 1, Symbol: "X", Side: Buy, Price: 10, Qty: 0}},
		{"negative qty", Order{ID: 1, Symbol: "X", Side: Buy, Price: 10, Qty: -1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.Process(tc.order)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

// ─── no match – order rests in book ────────────────────────────────────────

func TestProcess_NoMatch_BuyRests(t *testing.T) {
	e := New()
	trades := mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 5})
	requireTrades(t, trades, 0)
}

func TestProcess_NoMatch_SellRests(t *testing.T) {
	e := New()
	trades := mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 110, Qty: 5})
	requireTrades(t, trades, 0)
}

// ─── full fill ──────────────────────────────────────────────────────────────

func TestProcess_FullFill_BuyAgainstSell(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 10})

	trades := mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 105, Qty: 10})
	requireTrades(t, trades, 1)

	tr := trades[0]
	if tr.BuyOrderID != 2 || tr.SellOrderID != 1 {
		t.Errorf("wrong order IDs: %+v", tr)
	}
	if tr.Price != 100 {
		t.Errorf("trade price should be ask price 100, got %d", tr.Price)
	}
	if tr.Qty != 10 {
		t.Errorf("trade qty should be 10, got %d", tr.Qty)
	}
	if tr.Symbol != "AAPL" {
		t.Errorf("unexpected symbol %q", tr.Symbol)
	}
}

func TestProcess_FullFill_SellAgainstBuy(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Buy, Price: 110, Qty: 8})

	trades := mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 8})
	requireTrades(t, trades, 1)

	tr := trades[0]
	if tr.BuyOrderID != 1 || tr.SellOrderID != 2 {
		t.Errorf("wrong order IDs: %+v", tr)
	}
	if tr.Price != 110 {
		t.Errorf("trade price should be bid price 110, got %d", tr.Price)
	}
	if tr.Qty != 8 {
		t.Errorf("trade qty should be 8, got %d", tr.Qty)
	}
}

// ─── partial fill ───────────────────────────────────────────────────────────

func TestProcess_PartialFill_IncomingSmaller(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 10})

	// Buy 4 against the resting sell of 10 – sell order partially fills.
	trades := mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 4})
	requireTrades(t, trades, 1)
	if trades[0].Qty != 4 {
		t.Errorf("expected tradedQty=4, got %d", trades[0].Qty)
	}

	// A further buy of 6 should clear the remaining resting qty.
	trades = mustProcess(t, e, Order{ID: 3, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 6})
	requireTrades(t, trades, 1)
	if trades[0].Qty != 6 {
		t.Errorf("expected tradedQty=6, got %d", trades[0].Qty)
	}
}

func TestProcess_PartialFill_IncomingLarger(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 5})

	// Buy 10 – should fill the resting 5 and rest the remaining 5 in the book.
	trades := mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 10})
	requireTrades(t, trades, 1)
	if trades[0].Qty != 5 {
		t.Errorf("expected tradedQty=5, got %d", trades[0].Qty)
	}

	// A new sell should match the resting 5 from the previous buy.
	trades = mustProcess(t, e, Order{ID: 3, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 5})
	requireTrades(t, trades, 1)
	if trades[0].Qty != 5 {
		t.Errorf("expected tradedQty=5, got %d", trades[0].Qty)
	}
}

// ─── multiple levels consumed ────────────────────────────────────────────

func TestProcess_SweepMultiplePriceLevels(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "X", Side: Sell, Price: 100, Qty: 3})
	mustProcess(t, e, Order{ID: 2, Symbol: "X", Side: Sell, Price: 101, Qty: 3})
	mustProcess(t, e, Order{ID: 3, Symbol: "X", Side: Sell, Price: 102, Qty: 3})

	// Buy at 102 sweeps all three levels.
	trades := mustProcess(t, e, Order{ID: 4, Symbol: "X", Side: Buy, Price: 102, Qty: 9})
	requireTrades(t, trades, 3)

	prices := []int{100, 101, 102}
	for i, tr := range trades {
		if tr.Price != prices[i] {
			t.Errorf("trade[%d] price: want %d, got %d", i, prices[i], tr.Price)
		}
		if tr.Qty != 3 {
			t.Errorf("trade[%d] qty: want 3, got %d", i, tr.Qty)
		}
	}
}

// ─── FIFO (price-time priority) ──────────────────────────────────────────

func TestProcess_FIFOWithinPriceLevel(t *testing.T) {
	e := New()
	// Three sell orders at the same price – should be matched in arrival order.
	mustProcess(t, e, Order{ID: 10, Symbol: "Z", Side: Sell, Price: 50, Qty: 1})
	mustProcess(t, e, Order{ID: 11, Symbol: "Z", Side: Sell, Price: 50, Qty: 1})
	mustProcess(t, e, Order{ID: 12, Symbol: "Z", Side: Sell, Price: 50, Qty: 1})

	buy := Order{ID: 99, Symbol: "Z", Side: Buy, Price: 50, Qty: 3}
	trades := mustProcess(t, e, buy)
	requireTrades(t, trades, 3)

	wantSellIDs := []int64{10, 11, 12}
	for i, tr := range trades {
		if tr.SellOrderID != wantSellIDs[i] {
			t.Errorf("trade[%d]: expected SellOrderID=%d, got %d", i, wantSellIDs[i], tr.SellOrderID)
		}
	}
}

// ─── no cross-symbol matching ───────────────────────────────────────────

func TestProcess_SymbolIsolation(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 5})

	// Buy for MSFT should not match AAPL's sell.
	trades := mustProcess(t, e, Order{ID: 2, Symbol: "MSFT", Side: Buy, Price: 100, Qty: 5})
	requireTrades(t, trades, 0)
}

// ─── trade price is resting-order price ─────────────────────────────────

func TestProcess_TradePriceIsRestingPrice(t *testing.T) {
	e := New()

	// Aggressive buy at 120, resting ask at 100.
	mustProcess(t, e, Order{ID: 1, Symbol: "T", Side: Sell, Price: 100, Qty: 5})
	trades := mustProcess(t, e, Order{ID: 2, Symbol: "T", Side: Buy, Price: 120, Qty: 5})
	requireTrades(t, trades, 1)
	if trades[0].Price != 100 {
		t.Errorf("trade price should be resting ask 100, got %d", trades[0].Price)
	}

	// Aggressive sell at 80, resting bid at 110.
	e2 := New()
	mustProcess(t, e2, Order{ID: 3, Symbol: "T", Side: Buy, Price: 110, Qty: 5})
	trades = mustProcess(t, e2, Order{ID: 4, Symbol: "T", Side: Sell, Price: 80, Qty: 5})
	requireTrades(t, trades, 1)
	if trades[0].Price != 110 {
		t.Errorf("trade price should be resting bid 110, got %d", trades[0].Price)
	}
}

// ─── cancel ─────────────────────────────────────────────────────────────

func TestCancel_RemovesOrderFromBook(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 5})

	if err := e.Cancel(1); err != nil {
		t.Fatalf("Cancel(1) unexpected error: %v", err)
	}

	// After cancellation the book should be empty; a buy should not match.
	trades := mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 5})
	requireTrades(t, trades, 0)
}

func TestCancel_ReturnsErrorForUnknownOrder(t *testing.T) {
	e := New()
	if err := e.Cancel(999); err == nil {
		t.Error("expected error when cancelling unknown order, got nil")
	}
}

func TestCancel_ReturnsErrorAfterFill(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 5})
	mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 5})

	// Order 1 was fully filled; cancelling it should fail.
	if err := e.Cancel(1); err == nil {
		t.Error("expected error when cancelling filled order, got nil")
	}
}

func TestCancel_ZeroID(t *testing.T) {
	e := New()
	if err := e.Cancel(0); err == nil {
		t.Error("expected error for zero ID, got nil")
	}
}

func TestCancel_PartialFillThenCancel(t *testing.T) {
	e := New()
	mustProcess(t, e, Order{ID: 1, Symbol: "AAPL", Side: Sell, Price: 100, Qty: 10})

	// Partially fill order 1.
	mustProcess(t, e, Order{ID: 2, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 4})

	// Cancel the remainder.
	if err := e.Cancel(1); err != nil {
		t.Fatalf("Cancel(1) unexpected error: %v", err)
	}

	// Book should now be empty on the ask side.
	trades := mustProcess(t, e, Order{ID: 3, Symbol: "AAPL", Side: Buy, Price: 100, Qty: 10})
	requireTrades(t, trades, 0)
}

// ─── Trade.String() ──────────────────────────────────────────────────────

func TestTradeString(t *testing.T) {
	tr := Trade{BuyOrderID: 1, SellOrderID: 2, Symbol: "AAPL", Price: 100, Qty: 5}
	s := tr.String()
	if s == "" {
		t.Error("Trade.String() returned empty string")
	}
	want := "TRADE symbol=AAPL price=100 qty=5 buy=1 sell=2"
	if s != want {
		t.Errorf("Trade.String() = %q, want %q", s, want)
	}
}
