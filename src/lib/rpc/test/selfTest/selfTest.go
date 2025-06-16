package main

import (
	"context"
	"fmt"
	"local/lib/finalizer"
	"local/lib/rpc"
	"local/lib/transport/types"
	"os"
	"strconv"
	"sync"
)

var tc int
var mx sync.Mutex

func test() {
	mx.Lock()
	tc += 1
	mx.Unlock()
}

func testC() {
	rpc.FunctionClientStub[any](addr, "test", "")
}

func testR(a []byte) []byte {
	return rpc.FunctionServerStub[string](a, func(as string) any {
		test()
		return nil
	})
}

func main() {
	ctx, cancel := finalizer.WithCancel(context.Background())
	defer func() { cancel(); <-ctx.Done() }()
	rpc.Start(ctx)
	rpc.RegisterFunc("test", testR)
	addr = rpc.LocalAddr()
	c, e := strconv.Atoi(os.Args[1])
	if e != nil {
		panic(e)
	}
	var wg sync.WaitGroup
	for range c {
		wg.Add(1)
		go func() {
			testC()
			wg.Done()
		}()
	}
	wg.Wait()
	if tc != c {
		panic(fmt.Errorf("count mismatch %d %d", tc, c))
	}
}

var addr types.NetAddr
