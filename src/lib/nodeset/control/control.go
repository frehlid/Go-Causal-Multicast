package control

import (
	"local/lib/finalizer"
	"local/lib/nodeset"
	client "local/lib/nodeset/rpc/clientStub"
	server "local/lib/nodeset/server/rpc/clientStub"
	"local/lib/nodeset/server/types"
	"local/lib/rpc"
)

func Bind(addr string) {
	server.Bind(addr)
}

func Add(ctx *finalizer.Context, name string) *nodeset.Group {
	g := nodeset.NewGroup(name)
	finalizer.AfterFunc(ctx, func() {
		Remove(g)
	})
	g.SetNodeId(server.Add(g.Name, rpc.LocalAddr().String()))
	return g
}

func Get(g *nodeset.Group) []types.Node {
	return server.Get(g.Name)
}

func Remove(g *nodeset.Group) {
	server.Remove(g.Name, g.NodeId())
}

func AwaitSize(g *nodeset.Group, size int) {
	nodeset.AwaitSizeAtNode(g.Name, size)
	id := g.NodeId()
	for _, node := range g.Nodeset() {
		if node.Id != id {
			client.AwaitSizeAtNode(g.Name, node.Addr, size)
		}
	}
}

type ChangeListener int
