package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"local/lib/transport/types"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// initiate new call and return result or error
func Call(payload *bytes.Buffer, to types.NetAddr) (result *bytes.Buffer, err error) {
	seq := hostSeq.Add(1)
	id := callId{hostId, seq, call}
	res := make(chan *bytes.Buffer, 1)
	mx.Lock()
	pendingCalls[seq] = res
	mx.Unlock()
	defer func() {
		mx.Lock()
		delete(pendingCalls, seq)
		mx.Unlock()
	}()

	err = send(id, call, payload.Bytes(), to)
	if err == nil {
		defer func() {
			mx.Lock()
			delete(awaitingAck, id)
			mx.Unlock()
		}()
		for {
			mx.Lock()
			acked := awaitingAck[id]
			if acked == nil {
				acked = make(chan bool)
				awaitingAck[id] = acked
			}
			mx.Unlock()
			timeout := make(chan bool)
			go func() {
				time.Sleep(responseTimeout)
				timeout <- true
			}()
			select {
			case result = <-res:
				return
			case <-acked:
				continue
			case <-timeout:
				err = errors.New("reply timeout")
				return
			}
		}
	}
	return
}

// start listenting for incoming calls
func Listen(context context.Context, handleCall func(msg *bytes.Buffer, from types.NetAddr) []byte) {
	ctx = context
	go func() {
		for {
			bufData := make([]byte, bufferSize)
			n, from, err := conn.ReadFromUDP(bufData)
			if err != nil {
				panic(err)
			}
			bufData = bufData[:n]
			buf := bytes.NewBuffer(bufData)
			var id callId
			var tp byte
			binary.Read(buf, binary.NativeEndian, &id)
			binary.Read(buf, binary.NativeEndian, &tp)

			switch tp {
			case call:

				if !isDuplicateCall(id.HostId, id.Seq) {
					go func() {
						done := make(chan bool, 1)
						go func() {
							iteration := 0
							for {
								if iteration < ackIntervalCount {
									time.Sleep(ackInterval)
									iteration += 1
								} else {
									time.Sleep(responseInterval)
								}
								select {
								case <-ctx.Done():
									return
								case <-done:
									return
								default:
									send(id, ack, nil, from)
								}
							}
						}()
						res := handleCall(buf, from)
						done <- true
						send(callId{id.HostId, id.Seq, result}, result, res, from)
					}()
				} else {
					send(id, ack, nil, from)
				}

			case result:
				go send(id, ack, nil, from)
				mx.Lock()
				cid := callId{id.HostId, id.Seq, call}
				ackCh := awaitingAck[cid]
				delete(awaitingAck, cid)
				result := pendingCalls[id.Seq]
				mx.Unlock()
				if ackCh != nil {
					ackCh <- true
				}
				if result != nil {
					select {
					case result <- buf:
					default:
					}
				}

			case ack:
				mx.Lock()
				ackCh := awaitingAck[id]
				delete(awaitingAck, id)
				mx.Unlock()
				if ackCh != nil {
					ackCh <- true
				}
			}
		}
	}()
}

// return the local network address
func LocalAddr() types.NetAddr {
	return conn.LocalAddr()
}

func ResolveNetAddr(addr string) types.NetAddr {
	netAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	return netAddr
}

func send(id callId, tp byte, payload []byte, to types.NetAddr) (err error) {

	buf := bytes.NewBuffer(make([]byte, 0, bufferSize))
	binary.Write(buf, binary.NativeEndian, id)
	binary.Write(buf, binary.NativeEndian, tp)
	binary.Write(buf, binary.NativeEndian, payload)

	var acked chan bool
	if tp != ack {
		mx.Lock()
		acked = awaitingAck[id]
		if acked == nil {
			acked = make(chan bool)
			awaitingAck[id] = acked
		}
		mx.Unlock()
		defer func() {
			mx.Lock()
			delete(awaitingAck, id)
			mx.Unlock()
		}()
	}

	for range ackTimeoutCount {
		var n int
		n, err = conn.WriteTo(buf.Bytes(), to)
		if err != nil {
			return
		}
		if n != buf.Len() {
			err = fmt.Errorf("send truncated %d %d", n, buf.Len())
			return
		}
		if tp == ack {
			return
		}
		timeout := make(chan bool)
		go func() {
			time.Sleep(ackTimeout)
			timeout <- true
		}()
		select {
		case <-acked:
			return
		case <-timeout:
		}
	}
	err = fmt.Errorf("send timeout: %d %d %s", tp, id.Seq, to)

	return
}

func isDuplicateCall(hostId uint32, seq uint32) bool {
	now := time.Now()
	mx.Lock()
	defer mx.Unlock()

	hs := hostStates[hostId]
	if hs == nil {
		hs = &hostState{0, make(map[uint32]bool), now}
		hs.staleSeqs[seq] = true
		hostStates[hostId] = hs
		return false
	}

	if seq < hs.minFreshSeq || hs.staleSeqs[seq] {
		return true
	} else {

		hs.staleSeqs[seq] = true
		hs.lastActive = now

		for {
			if hs.staleSeqs[hs.minFreshSeq] {
				delete(hs.staleSeqs, hs.minFreshSeq)
				hs.minFreshSeq += 1
			} else {
				break
			}
		}

		hostStateGCCountdown -= 1
		if hostStateGCCountdown == 0 {
			hostStateGCCountdown = hostStateGCFreq
			for hostId, hs := range hostStates {
				if now.Sub(hs.lastActive) > forgetHostTimeout {
					delete(hostStates, hostId)
				}
			}
		}

		return false
	}
}

const useLocalHost = false

func init() {
	mx.Lock()
	defer mx.Unlock()
	if conn == nil {
		var ip string
		if useLocalHost {
			ip = "127.0.0.1:0"
		} else {
			addrs, err := net.InterfaceAddrs()
			if err != nil {
				panic(err)
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					ip = ipNet.IP.String()
				}
			}
			ip += ":0"
		}
		addr, err := net.ResolveUDPAddr("udp", ip)
		if err != nil {
			panic(err)
		}

		conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			panic(err)
		}
	}
}

type hostState struct {
	minFreshSeq uint32
	staleSeqs   map[uint32]bool
	lastActive  time.Time
}

type callId struct {
	HostId uint32
	Seq    uint32
	Tp     byte
}

const (
	call byte = iota
	result
	ack
)
const hostStateGCFreq = 50
const forgetHostTimeout = time.Duration(24 * time.Hour)
const bufferSize = 2048
const maxLatency = 500 * time.Millisecond
const ackInterval = maxLatency
const ackIntervalCount = 8
const ackTimeout = ackInterval * 2
const ackTimeoutCount = ackIntervalCount / 2
const responseInterval = 2 * time.Second
const responseTimeout = responseInterval * 4

var hostStates = make(map[uint32]*hostState)
var hostStateGCCountdown = hostStateGCFreq
var conn *net.UDPConn = nil
var ctx context.Context = context.TODO()
var mx sync.Mutex
var pendingCalls = make(map[uint32]chan *bytes.Buffer)
var awaitingAck = make(map[callId]chan bool)
var hostId = rand.Uint32()
var hostSeq atomic.Uint32
