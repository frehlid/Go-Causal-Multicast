package clientStub

import (
	nodesetServerStub "local/lib/nodeset/rpc/serverStub"
	"local/lib/nodeset/server/rpc/api"
	"local/lib/nodeset/server/types"
	"local/lib/rpc"
	"local/lib/transport"
	transportTypes "local/lib/transport/types"
)

func Bind(addr string) {
	server = transport.ResolveNetAddr(addr)
	nodesetServerStub.Register()
}

func Add(groupName string, addr string) uint32 {
	return rpc.FunctionClientStub[uint32](server, api.Add, api.AddArgs{GroupName: groupName, Addr: addr})
}

func Remove(groupName string, id uint32) {
	rpc.FunctionClientStub[any](server, api.Remove, api.RemoveArgs{GroupName: groupName, Id: id})
}

func Get(groupName string) []types.Node {
	return rpc.FunctionClientStub[[]types.Node](server, api.Get, api.GetArgs{GroupName: groupName})
}

var server transportTypes.NetAddr
