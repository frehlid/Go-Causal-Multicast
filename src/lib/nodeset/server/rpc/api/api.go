package api

import (
	"local/lib/nodeset/server/types"
)

const Add = "Add"
const Remove = "Remove"
const Get = "Get"
const ChangeListener_Changed = "ChangeListener_Changed"

type AddArgs struct {
	GroupName string
	Addr      string
}

type RemoveArgs struct {
	GroupName string
	Id        uint32
}

type GetArgs struct {
	GroupName string
}

type ChangeListener_ChangedArgs = struct {
	Nodeset []types.Node
}
