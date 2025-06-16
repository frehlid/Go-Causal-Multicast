package serverStub

import (
	"local/chat"
	"local/chat/rpc/api"
	"local/lib/rpc"
)

func Register() {
	rpc.RegisterFunc(api.Display, Display)
}

func Display(argData []byte) []byte {
	return rpc.FunctionServerStub[api.DisplayArgs](argData, func(args api.DisplayArgs) any {
		chat.Display(args.Msg)
		return nil
	})
}
