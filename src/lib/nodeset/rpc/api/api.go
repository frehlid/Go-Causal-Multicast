package api

import "local/lib/nodeset/server/types"

const AwaitSizeAtNode = "AwaitSizeAtNode"
const Update = "Update"

type AwaitSizeAtNodeArgs struct {
	Group string
	Size  int
}

type UpdateArgs struct {
	Group   string
	Nodeset []types.Node
}
