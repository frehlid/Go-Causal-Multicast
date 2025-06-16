package main

import (
	"context"
	"fmt"
	"os"

	"local/lib/finalizer"
	"local/lib/rpc"
	"local/lib/transport"
	"local/lib/transport/types"
)

func Test(i int, s string) string {
	return fmt.Sprintf("%d %s", i, s)
}

const TestName = "Test"
const GetRRName = "GetRR"

type TestArgs struct {
	I int
	S string
}
type GetRRArgs struct{}

func TestServerStub(argData []byte) []byte {
	return rpc.FunctionServerStub[TestArgs](argData, func(args TestArgs) any {
		return Test(args.I, args.S)
	})
}

type obj int

var anObj obj = 100

func (o obj) TestMethod(i int, s string) string {
	return fmt.Sprintf("%d %d %s", o, i, s)
}

func TestClientStub(server types.NetAddr, i int, s string) string {
	return rpc.FunctionClientStub[string](server, TestName, TestArgs{I: i, S: s})
}

func GetRR() *obj {
	return &anObj
}

type objProxy rpc.Proxy

const TestMethodName = "TestMethod"

type TestMethodArgs struct {
	I int
	S string
}

func (o objProxy) TestMethodClientStub(i int, s string) string {
	return rpc.MethodClientStub[string](o.RemoteReference, TestMethodName, TestMethodArgs{I: i, S: s})
}

func TestMethodServerStub(receiver any, argData []byte) []byte {
	return rpc.MethodServerStub[*obj, TestMethodArgs](receiver.(*obj), argData, func(o *obj, a TestMethodArgs) any {
		return o.TestMethod(a.I, a.S)
	})
}

func GetRRClientStub(server types.NetAddr) objProxy {
	return rpc.GetProxy[objProxy](rpc.FunctionClientStub[rpc.RemoteReference](server, GetRRName, GetRRArgs{}))
}

func getRRServerStub(argData []byte) []byte {
	return rpc.FunctionServerStub[GetRRArgs](argData, func(args GetRRArgs) any {
		return rpc.GetRemoteReference(GetRR())
	})
}

func main() {
	ctx, _ := finalizer.WithCancel(context.Background())
	rpc.Start(ctx)
	fmt.Println(rpc.LocalAddr().String())
	if os.Args[1] == "s" {
		rpc.RegisterFunc(TestName, TestServerStub)
		rpc.RegisterFunc(GetRRName, getRRServerStub)
		rpc.RegisterMethod(TestMethodName, TestMethodServerStub)
	} else {
		server := transport.ResolveNetAddr(os.Args[2])
		fmt.Println(TestClientStub(server, 10, "hello world"))
		rr := GetRRClientStub(server)
		fmt.Println(rr.TestMethodClientStub(20, "goodbye world"))
	}
	<-ctx.Done()
}
