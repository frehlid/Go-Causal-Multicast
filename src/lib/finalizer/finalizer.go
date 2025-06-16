package finalizer

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Cancel context that ensures that AfterFunc is called in event of ctrl-c or runtime error
//
// Add this to the beginning of main()
//		ctx, cancel := finalizer.WithCancel(context.Background())
//		defer func() { cancel(); <-ctx.Done() }()
// And this to the end
//		<-ctx.Done()

type Context struct {
	ctx                context.Context
	afterFuncWaitGroup sync.WaitGroup
}

func (f *Context) Deadline() (deadline time.Time, ok bool) {
	return f.ctx.Deadline()
}

func (f *Context) Done() <-chan struct{} {
	select {
	case <-f.ctx.Done():
		f.afterFuncWaitGroup.Wait()
		return f.ctx.Done()
	default:
		ch := make(chan struct{})
		go func() {
			<-f.ctx.Done()
			f.afterFuncWaitGroup.Wait()
			close(ch)
		}()
		return ch
	}
}

func (f *Context) Err() error {
	return f.ctx.Err()
}

func (f *Context) Value(key any) any {
	return f.ctx.Value(key)
}

func WithCancel(parentContext context.Context) (ctx *Context, cancel func()) {
	ctx = &Context{}
	ctx.ctx, cancel = context.WithCancel(parentContext)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		signal.Reset()
		cancel()
	}()
	return
}

func AfterFunc(ctx *Context, f func()) {
	ctx.afterFuncWaitGroup.Add(1)
	context.AfterFunc(ctx.ctx, func() {
		f()
		ctx.afterFuncWaitGroup.Done()
	})
}
