package clientStub

import (
	"local/lib/nodeset/rpc/api"
	"local/lib/nodeset/server/types"
	"local/lib/rpc"
	"local/lib/transport"
)

func AwaitSizeAtNode(group string, node string, size int) {
	rpc.FunctionClientStub[any](transport.ResolveNetAddr(node), api.AwaitSizeAtNode, api.AwaitSizeAtNodeArgs{Group: group, Size: size})
}

func Update(group string, node string, nodeset []types.Node) (err error) {
	_, err = rpc.FunctionClientStubWithError[any](transport.ResolveNetAddr(node), api.Update, api.UpdateArgs{Group: group, Nodeset: nodeset})
	return
}
