all: main.wasm wasm_exec.js
ifeq (, $(shell which goeval))
	go install github.com/dolmen-go/goeval@latest
endif
	goeval 'http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`.`)))'

wasm_exec.js:
	cp "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" .

main.wasm: wasm_main.go
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o main.wasm