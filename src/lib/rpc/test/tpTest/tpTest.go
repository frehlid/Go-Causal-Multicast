package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"

	"local/lib/finalizer"
	"local/lib/transport"
	"local/lib/transport/types"
)

func main() {
	ctx, _ := finalizer.WithCancel(context.TODO())
	go transport.Listen(ctx, func(msg *bytes.Buffer, from types.NetAddr) []byte {
		// time.Sleep(100 * time.Second)
		fmt.Println("handle")
		return []byte("return value")
	})
	if os.Args[1] == "s" {
		fmt.Println(transport.LocalAddr())
	} else {

		// addr := transport.ResolveNetAddr(os.Args[2])
		// msg := bytes.NewBuffer([]byte("foo"))
		// _, e := transport.Call(ctx, msg, addr)
		// if e {
		// 	panic("error")
		// }

		var wg sync.WaitGroup
		wg.Add(100)
		for range 100 {
			go func() {
				addr := transport.ResolveNetAddr(os.Args[2])
				msg := bytes.NewBuffer([]byte("foo"))
				transport.Call(msg, addr)
				wg.Done()
			}()
		}
		wg.Wait()
		fmt.Println("done")
	}
	<-ctx.Done()
}
