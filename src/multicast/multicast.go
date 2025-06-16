package multicast

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"local/lib/nodeset"
	"local/lib/transport/types"
	"sync"
	"time"
)

type TimeVector []uint32

type TimeVectorArgs struct {
	IncomingTimestamp TimeVector
	GroupName         string
}

type scheduledHandleCall struct {
	payload    *bytes.Buffer
	fromAddr   types.NetAddr
	handleCall func(*bytes.Buffer, types.NetAddr) []byte
}

func NewGroup(nodeset *nodeset.Group) *Group {
	// TODO: this makes the clock have dead weight (leaving and rejoining)
	highestId := nodeset.Nodeset()[nodeset.Size()-1].Id
	g := &Group{name: nodeset.Name, Nodeset: nodeset, Clock: make(TimeVector, highestId+1), highestIdSeen: highestId, AllIdsSeen: make(map[string]uint32), scheduleQueue: make(chan scheduledHandleCall, 512)}
	groups[nodeset.Name] = g

	// store these so if a client leaves the nodeset and their message gets delivered after, we still know what their id was
	for _, node := range nodeset.Nodeset() {
		g.AllIdsSeen[node.Addr] = node.Id
	}

	g.Nodeset.AddChangeListener(func() {
		newHighestId := g.Nodeset.Nodeset()[g.Nodeset.Size()-1].Id
		if newHighestId > g.highestIdSeen {
			g.Clock = append(g.Clock, 0)
			g.highestIdSeen = newHighestId

			g.GroupMutex.Lock()
			g.AllIdsSeen[g.Nodeset.Nodeset()[nodeset.Size()-1].Addr] = newHighestId // store this id so if they leave we can still access it
			g.GroupMutex.Unlock()
		}
	})
	go g.schedulerThread()
	return g
}

// thread that uses a channel to handle invoking handleCall in FIFO order (but outside the critical section in schedule delivery)
func (g *Group) schedulerThread() {
	for scheduledTask := range g.scheduleQueue {
		scheduledTask.handleCall(scheduledTask.payload, scheduledTask.fromAddr)
	}
}

func ScheduleDelivery(handleCall func(buf *bytes.Buffer, fromAddr types.NetAddr) []byte, buf *bytes.Buffer, fromAddr types.NetAddr) []byte {
	funcName, payloadLen := PeekFuncName(buf)

	if funcName == "Display" {
		bufferCopy := bytes.NewBuffer(buf.Bytes())
		handleDisplayMessage(handleCall, bufferCopy, fromAddr, payloadLen)
		return nil
	}
	return handleCall(buf, fromAddr)
}

func handleDisplayMessage(handleCall func(buf *bytes.Buffer, fromAddr types.NetAddr) []byte, buf *bytes.Buffer, fromAddr types.NetAddr, payloadLen int32) []byte {
	payload, incomingTimestamp, groupName := getGroupInformation(buf, payloadLen)
	nodeId := getIdFromAddr(groups[groupName], fromAddr)

	group := groups[groupName]

	for {
		group.GroupMutex.Lock()
		validTimestamp := compareTimestamps(incomingTimestamp, group.Clock, nodeId)
		if validTimestamp {
			//	fmt.Println("Delivering msg: ", payload, " incoming timestamp: ", incomingTimestamp, " local timestamp: ", group.Clock)
			for k := 0; k < len(incomingTimestamp); k++ {
				group.Clock[k] = max(group.Clock[k], incomingTimestamp[k])
			}

			// instead of calling this inside the critical section, push it to the queue and let the scheduler thread call it
			callToSchedule := scheduledHandleCall{payload: payload, fromAddr: fromAddr, handleCall: handleCall}
			group.scheduleQueue <- callToSchedule
			group.GroupMutex.Unlock()

			return nil
		}
		group.GroupMutex.Unlock()
		time.Sleep(100 * time.Millisecond)
	}
}

func compareTimestamps(incomingTimestamp, localTimestamp TimeVector, nodeId uint32) bool {
	// conditions needed to deliver message:
	// 1. incoming timestamp is exactly one more than our local timestamp
	validTimestamp := false
	if incomingTimestamp[nodeId] == localTimestamp[nodeId]+1 {
		// 2.  every other time is less than or equal to your own timestamp value
		validTimestamp = true
		for i := 0; i < len(incomingTimestamp); i++ {
			if uint32(i) != nodeId && incomingTimestamp[i] > localTimestamp[i] {
				validTimestamp = false
				break
			}
		}
	}
	return validTimestamp
}

func getIdFromAddr(group *Group, fromAddr types.NetAddr) uint32 {
	id, exists := group.AllIdsSeen[fromAddr.String()]
	if exists {
		return id
	}
	panic("no such node")
}

func PeekFuncName(buf *bytes.Buffer) (string, int32) {
	// todo actually learn how to use buffers or marshall ur shit in a non lazy way dumbass
	bufferCopy := bytes.NewBuffer(buf.Bytes())
	reader := bufio.NewReader(bufferCopy)

	// skip the first byte (indicates call or result or ack)
	_, err := reader.ReadByte()
	if err != nil {
		panic(err)
	}

	var nameLen int32
	err = binary.Read(reader, binary.NativeEndian, &nameLen)
	if err != nil {
		panic(err)
	}

	nameBytes := make([]byte, nameLen)
	_, err = io.ReadFull(reader, nameBytes)
	if err != nil {
		panic(err)
	}

	var argLen int32
	err = binary.Read(reader, binary.NativeEndian, &argLen)
	if err != nil {
		panic(err)
	}

	return string(nameBytes), argLen + nameLen + LENGTH_BYTES + EXTRA_JSON_BYTES
}

func getGroupInformation(buf *bytes.Buffer, payloadLength int32) (*bytes.Buffer, TimeVector, string) {
	// extract payload first
	payloadBytes := make([]byte, payloadLength)
	err := binary.Read(buf, binary.NativeEndian, &payloadBytes)
	if err != nil {
		panic(err)
	}

	var groupInformationArgs TimeVectorArgs
	err = json.Unmarshal(buf.Bytes(), &groupInformationArgs)
	if err != nil {
		panic(err)
	}

	return bytes.NewBuffer(payloadBytes), groupInformationArgs.IncomingTimestamp, groupInformationArgs.GroupName
}

type Group struct {
	name          string
	Nodeset       *nodeset.Group
	Clock         TimeVector
	highestIdSeen uint32
	AllIdsSeen    map[string]uint32
	GroupMutex    sync.Mutex
	scheduleQueue chan scheduledHandleCall
}

const LENGTH_BYTES = 2
const EXTRA_JSON_BYTES = 7

// this is the list of multicast groups that I am apart of...
var groups = make(map[string]*Group)
