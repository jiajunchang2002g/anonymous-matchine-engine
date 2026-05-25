package engine

import "fmt"

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Process(msg string) {
	fmt.Println("[engine]", msg)
}