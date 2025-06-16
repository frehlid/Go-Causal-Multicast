package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"local/lib/transport/types"
	"local/multicast"
	"local/multicast/transport"
	"math/rand/v2"
)

func Start(ctx context.Context) {
	transport.Listen(ctx, func(msg *bytes.Buffer, from types.NetAddr) []byte {
		var tp byte
		binary.Read(msg, binary.NativeEndian, &tp)
		switch tp {

		case call:
			funcName := string(readByteSlice(msg))
			args := readByteSlice(msg)
			handler := funcHandlers[funcName]
			if handler == nil {
				panic(fmt.Sprintf("unregistered rpc func: %s\n", funcName))
			}
			return handler(args)

		case invoke:
			var receiverGid globalObjectId
			binary.Read(msg, binary.NativeEndian, &receiverGid)
			methodName := string(readByteSlice(msg))
			args := readByteSlice(msg)
			receiver := receiverForGid[receiverGid]
			if receiver == nil {
				panic(fmt.Sprintf("Unknown gid %d for call to %s\n", receiverGid, methodName))
			}
			handler := receiverFuncHandlers[methodName]
			if handler == nil {
				panic(fmt.Sprintf("unregisred rpc method: %s\n", methodName))
			}
			return handler(receiver, args)

		}
		return []byte("")
	})
}

func Call(to types.NetAddr, funcName string, args []byte) (result []byte, err error) {
	payload := bytes.NewBuffer(make([]byte, 0))
	binary.Write(payload, binary.NativeEndian, call)
	writeByteSlice(payload, []byte(funcName))
	writeByteSlice(payload, args)
	r, err := transport.Call(payload, to)
	if err == nil {
		result = r.Bytes()
	}
	return
}

func Multicast(g *multicast.Group, funcName string, args []byte) (result []byte) {
	payload := bytes.NewBuffer(make([]byte, 0))
	binary.Write(payload, binary.NativeEndian, call)
	writeByteSlice(payload, []byte(funcName))
	writeByteSlice(payload, args)
	result = transport.Multicast(payload, g).Bytes()
	return
}

func Invoke(receiver RemoteReference, methodName string, args []byte) (result []byte, err error) {
	payload := bytes.NewBuffer(make([]byte, 0))
	binary.Write(payload, binary.NativeEndian, invoke)
	binary.Write(payload, binary.NativeEndian, receiver.Gid)
	writeByteSlice(payload, []byte(methodName))
	writeByteSlice(payload, args)
	r, err := transport.Call(payload, receiver.Host)
	if err == nil {
		result = r.Bytes()
	}
	return
}

func RegisterFunc(funcName string, handler funcHandler) {
	funcHandlers[funcName] = handler
}

func RegisterMethod(funcName string, handler methodHandler) {
	receiverFuncHandlers[funcName] = handler
}

func FunctionClientStubWithError[R any](server types.NetAddr, funcName string, args any) (result R, err error) {
	argData, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	resData, err := Call(server, funcName, argData)
	if err == nil {
		json.Unmarshal(resData, &result)
	}
	return
}

func FunctionClientStub[R any](server types.NetAddr, funcName string, args any) (result R) {
	result, err := FunctionClientStubWithError[R](server, funcName, args)
	if err != nil {
		panic(err)
	}
	return
}

func MulticastClientStub[R any](g *multicast.Group, funcName string, args any) (result R) {
	argData, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	resData := Multicast(g, funcName, argData)
	json.Unmarshal(resData, &result)
	return
}

func FunctionServerStub[A any](argData []byte, call func(args A) any) []byte {
	var args A
	json.Unmarshal(argData, &args)
	res := call(args)
	resData, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	return resData
}

func MethodClientStubWithError[R any](receiver RemoteReference, methodName string, args any) (result R, err error) {
	argData, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	resData, err := Invoke(receiver, methodName, argData)
	if err == nil {
		json.Unmarshal(resData, &result)
	}
	return
}

func MethodClientStub[R any](receiver RemoteReference, methodName string, args any) (result R) {
	result, err := MethodClientStubWithError[R](receiver, methodName, args)
	if err != nil {
		panic(err)
	}
	return
}

func MethodServerStub[R any, A any](receiver R, argData []byte, invoke func(receiver R, args A) any) []byte {
	var args A
	json.Unmarshal(argData, &args)
	res := invoke(receiver, args)
	resData, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	return resData
}

func GetRemoteReference(receiver any) RemoteReference {
	remoteReference := remoteReferenceForReceiver[receiver]
	if remoteReference.Gid == 0 {
		remoteReference = RemoteReference{
			Host: transport.LocalAddr(),
			Gid:  globalObjectId(rand.Uint64()),
		}
		remoteReferenceForReceiver[receiver] = remoteReference
		receiverForGid[remoteReference.Gid] = receiver
	}
	return remoteReference
}

func GetProxy[P ~struct{ RemoteReference }](ref RemoteReference) P {
	proxy := proxyForGid[ref.Gid]
	if proxy == nil {
		proxy = &P{ref}
		proxyForGid[ref.Gid] = proxy
	}
	return *(proxy.(*P))
}

func LocalAddr() types.NetAddr {
	return transport.LocalAddr()
}

func ResolveNetAddr(addr string) (netAddr types.NetAddr) {
	return transport.ResolveNetAddr(addr)
}

type RemoteReference struct {
	Host types.NetAddr
	Gid  globalObjectId
}

func (r RemoteReference) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Host string
		Gid  globalObjectId
	}{Host: r.Host.String(), Gid: r.Gid})
}

func (r *RemoteReference) UnmarshalJSON(data []byte) error {
	var raw struct {
		Host string
		Gid  globalObjectId
	}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	r.Host = transport.ResolveNetAddr(raw.Host)
	r.Gid = raw.Gid
	return nil
}

type Proxy struct {
	RemoteReference
}

func writeByteSlice(buf *bytes.Buffer, bytes []byte) {
	var len int32 = int32(len(bytes))
	binary.Write(buf, binary.NativeEndian, len)
	binary.Write(buf, binary.NativeEndian, bytes)
}

func readByteSlice(buf *bytes.Buffer) (bytes []byte) {
	var len int32
	binary.Read(buf, binary.NativeEndian, &len)
	bytes = make([]byte, len)
	binary.Read(buf, binary.NativeEndian, &bytes)
	return
}

type funcHandler func(args []byte) []byte
type methodHandler func(receiver any, args []byte) []byte
type globalObjectId uint64

const (
	call byte = iota
	invoke
)

var funcHandlers = make(map[string]funcHandler)
var receiverFuncHandlers = make(map[string]methodHandler)
var receiverForGid = make(map[globalObjectId]any)
var remoteReferenceForReceiver = make(map[any]RemoteReference)
var proxyForGid = make(map[globalObjectId]any)
