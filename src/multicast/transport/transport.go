package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"local/lib/transport"
	"local/lib/transport/types"
	"local/multicast"
	"math/rand/v2"
	"time"
)

func Multicast(payload *bytes.Buffer, g *multicast.Group) *bytes.Buffer {
	funcName, _ := multicast.PeekFuncName(payload)

	if funcName == "Display" {
		buf := bytes.NewBuffer(make([]byte, 0))
		_, err := buf.Write(payload.Bytes())
		if err != nil {
			panic(err)
		}

		groupName := g.Nodeset.Name
		myId := g.Nodeset.NodeId()

		g.GroupMutex.Lock()
		g.Clock[myId]++
		g.GroupMutex.Unlock()

		timeStamp := g.Clock
		timeVectorArgs := multicast.TimeVectorArgs{
			IncomingTimestamp: timeStamp,
			GroupName:         groupName,
		}

		vectorArgsBuf, err := json.Marshal(timeVectorArgs)
		if err != nil {
			panic(err)
		}

		_, err = buf.Write(vectorArgsBuf)
		if err != nil {
			panic(err)
		}

		payload = buf
	}

	for _, node := range g.Nodeset.Nodeset() {
		if node.Id != g.Nodeset.NodeId() {
			go func() {
				if node.Addr == slowNode {
					time.Sleep(time.Duration(rand.Uint32()%5000) * time.Millisecond)
				}
				_, e := transport.Call(bytes.NewBuffer(payload.Bytes()), ResolveNetAddr(node.Addr))
				if e != nil {
					panic(e)
				}
			}()
		}
	}
	return bytes.NewBuffer(make([]byte, 0))
}

func Call(payload *bytes.Buffer, to types.NetAddr) (result *bytes.Buffer, err error) {
	result, err = transport.Call(payload, to)
	return
}

// multicast listen
func Listen(context context.Context, handleCall func(msg *bytes.Buffer, from types.NetAddr) []byte) {
	transport.Listen(context, func(msg *bytes.Buffer, from types.NetAddr) []byte {
		return multicast.ScheduleDelivery(handleCall, msg, from)
	})
}

func LocalAddr() types.NetAddr {
	return transport.LocalAddr()
}

func ResolveNetAddr(addr string) types.NetAddr {
	return transport.ResolveNetAddr(addr)
}

func SetSlowNode(node string) {
	slowNode = node
}

var slowNode string
