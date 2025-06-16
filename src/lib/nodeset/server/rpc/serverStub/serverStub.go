package serverStub

import (
	"local/lib/nodeset/server"
	"local/lib/nodeset/server/rpc/api"
	"local/lib/rpc"
)

func Register() {
	rpc.RegisterFunc(api.Add, Add)
	rpc.RegisterFunc(api.Remove, Remove)
	rpc.RegisterFunc(api.Get, Get)
}

func Add(argData []byte) []byte {
	return rpc.FunctionServerStub[api.AddArgs](argData, func(args api.AddArgs) any {
		return server.Add(args.GroupName, args.Addr)
	})
}

func Remove(argData []byte) []byte {
	return rpc.FunctionServerStub[api.RemoveArgs](argData, func(args api.RemoveArgs) any {
		server.Remove(args.GroupName, args.Id)
		return nil
	})
}

func Get(argData []byte) []byte {
	return rpc.FunctionServerStub[api.GetArgs](argData, func(args api.GetArgs) any {
		return server.Get(args.GroupName)
	})
}
