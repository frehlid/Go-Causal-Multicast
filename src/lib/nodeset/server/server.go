package server

import (
	"fmt"
	"sync"

	"local/lib/nodeset/rpc/clientStub"
	"local/lib/nodeset/server/types"
)

func Add(groupName string, addr string) uint32 {
	mx.Lock()
	g := groups[groupName]
	if g == nil {
		g = &group{groupName, make([]types.Node, 0), 0}
		groups[groupName] = g
	}
	id := g.nextId
	g.nextId += 1
	g.nodeset = append(g.nodeset, types.Node{Id: id, Addr: addr})
	mx.Unlock()
	notifyNodes(g)
	return id
}

func Remove(groupName string, id uint32) {
	mx.Lock()
	g := groups[groupName]
	changed := false
	if g != nil {
		for i, e := range g.nodeset {
			if e.Id == id {
				g.nodeset = append(g.nodeset[:i], g.nodeset[i+1:]...)
				changed = true
				break
			}
		}
		if len(g.nodeset) == 0 {
			g.nextId = 0
		}
		mx.Unlock()
		if changed {
			notifyNodes(g)
		}
	} else {
		mx.Unlock()
	}
}

func Get(groupName string) (nodeset []types.Node) {
	g := groups[groupName]
	if g != nil {
		nodeset = g.nodeset
	}
	return
}

func notifyNodes(g *group) {
	fmt.Println(g.name, g.nodeset)
	var remove []uint32
	for _, n := range g.nodeset {
		err := clientStub.Update(g.name, n.Addr, g.nodeset)
		if err != nil {
			remove = append(remove, n.Id)
			fmt.Printf("Node %s not responding; removed\n", n.Addr)
		}
	}
	for _, r := range remove {
		Remove(g.name, r)
	}
}

type group struct {
	name    string
	nodeset []types.Node
	nextId  uint32
}

var groups = make(map[string]*group)

var mx sync.Mutex
