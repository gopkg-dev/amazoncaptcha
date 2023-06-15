//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"syscall/js"

	"github.com/gopkg-dev/amazoncaptcha"
)

func SolveCaptcha(_ js.Value, args []js.Value) any {

	buffer := make([]byte, args[0].Length())
	js.CopyBytesToGo(buffer, args[0])

	solve, err := amazoncaptcha.Solve(bytes.NewReader(buffer))
	if err != nil {
		panic(err)
	}

	return solve
}

func main() {
	done := make(chan int, 0)
	js.Global().Set("SolveCaptcha", js.FuncOf(SolveCaptcha))
	<-done
}
