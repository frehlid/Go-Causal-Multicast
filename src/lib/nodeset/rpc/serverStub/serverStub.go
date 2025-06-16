package serverstub

import (
	"local/lib/nodeset"
	"local/lib/nodeset/rpc/api"
	"local/lib/rpc"
)

func Register() {
	rpc.RegisterFunc(api.AwaitSizeAtNode, AwaitSizeAtNode)
	rpc.RegisterFunc(api.Update, Update)

}

func AwaitSizeAtNode(argData []byte) []byte {
	return rpc.FunctionServerStub[api.AwaitSizeAtNodeArgs](argData, func(a api.AwaitSizeAtNodeArgs) any {
		nodeset.AwaitSizeAtNode(a.Group, a.Size)
		return nil
	})
}

func Update(argData []byte) []byte {
	return rpc.FunctionServerStub[api.UpdateArgs](argData, func(a api.UpdateArgs) any {
		nodeset.Update(a.Group, a.Nodeset)
		return nil
	})
}

func init() {
	Register()
}
