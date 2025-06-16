package nodeset

import (
	"math/rand/v2"
	"sync"

	"local/lib/nodeset/server/types"
)

type Group struct {
	Name      string
	Leader    types.Node
	mx        sync.Mutex
	cv        *sync.Cond
	nodeId    uint32
	nodeset   []types.Node
	listeners []func()
}

func NewGroup(groupName string) *Group {
	g := &Group{Name: groupName}
	g.cv = sync.NewCond(&g.mx)
	groups[groupName] = g
	return g
}

func (g *Group) Size() int {
	return len(g.nodeset)
}

func (g *Group) Nodeset() []types.Node {
	g.mx.Lock()
	var ns = make([]types.Node, len(g.nodeset))
	copy(ns, g.nodeset)
	g.mx.Unlock()
	return ns
}

func (g *Group) NodeId() uint32 {
	return g.nodeId
}

func (g *Group) SetNodeId(id uint32) {
	g.mx.Lock()
	g.nodeId = id
	g.mx.Unlock()
}

func (g *Group) AddChangeListener(onChange func()) {
	g.listeners = append(g.listeners, onChange)
}

type ChangeListener int

func Update(groupName string, ns []types.Node) {
	g := groups[groupName]
	if g != nil {
		g.mx.Lock()
		g.nodeset = ns
		g.Leader = ns[rand.Uint32()%uint32(len(ns))]
		g.cv.Broadcast()
		g.mx.Unlock()
		for _, l := range g.listeners {
			l()
		}
	}
}

func AwaitSizeAtNode(groupName string, size int) {
	g := groups[groupName]
	if g != nil {
		g.cv.L.Lock()
		for {
			if len(g.nodeset) >= size {
				break
			} else {
				g.cv.Wait()
			}
		}
		g.cv.L.Unlock()
	}
}

var groups = make(map[string]*Group)
