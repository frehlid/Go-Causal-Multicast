package main

import (
	"context"
	"encoding/json"
	"fmt"
	"local/chat"
	"local/chat/rpc/serverStub"
	"local/lib/finalizer"
	"local/lib/nodeset/control"
	"local/lib/rpc"
	"local/multicast"
	"local/multicast/transport"
	"os"
	"slices"
	"strconv"
)

func main() {
	testFileName := os.Args[2]

	// NOTE: the test data must include an entry for nodes, even if they make no writes. See testInputs/bt.json for an example
	rawData, err := os.ReadFile("./cmd/testChat/testInputs/" + testFileName + ".json")
	if err != nil {
		panic(err)
	}

	runTestCase(rawData)
}

func runTestCase(rawData []byte) {
	var testData map[string]map[string][]string
	err := json.Unmarshal(rawData, &testData)
	if err != nil {
		panic(err)
	}

	numNodes = len(testData) - 1
	numMessages := 0

	ctx, cancel := finalizer.WithCancel(context.Background())
	defer func() { cancel(); <-ctx.Done() }()
	control.Bind(os.Args[1])
	serverStub.Register()
	rpc.Start(ctx)

	g := multicast.NewGroup(control.Add(ctx, "chat"))
	// parse the inputs into ws
	for key, innerMap := range testData {
		if key == "slowChannels" {
			for slowKey, slowValues := range innerMap {
				slowFromChannel, parseErr := strconv.Atoi(slowKey)
				if err != nil {
					panic(parseErr)
				}

				slowToChannel, parseErr := strconv.Atoi(slowValues[0])
				if err != nil {
					panic(parseErr)
				}
				slowChannel := channel{from: uint32(slowFromChannel), to: uint32(slowToChannel)}
				slowChannels = append(slowChannels, slowChannel)
			}
		} else {
			nodeNum, parseErr := strconv.Atoi(key)
			if parseErr != nil {
				panic(parseErr)
			}
			for innerKey, values := range innerMap {
				if innerKey == "nil" {
					innerKey = ""
				}
				ws(uint32(nodeNum), innerKey, values...)
				numMessages += len(values)
			}

		}
	}
	displayCh := make(chan string)

	// goroutine that will write from output chanel to output array
	outChan := make(chan string, numMessages)
	doneChan := make(chan bool)
	go func() {
		for msg := range outChan {
			outputs = append(outputs, msg)

			// if we have received all the messages in the test case, stop and start testing
			if len(outputs) == numMessages {
				doneChan <- true
				close(outChan)
				return
			}
		}
	}()

	chat.SetDisplayFunc(func(msg string) {
		fmt.Println(msg)
		outChan <- msg
		displayCh <- msg
	})

	control.AwaitSize(g.Nodeset, numNodes)
	for _, c := range slowChannels {
		if c.from == g.Nodeset.NodeId() {
			transport.SetSlowNode(g.Nodeset.Nodeset()[c.to].Addr)
		}
	}

	for _, s := range script[g.Nodeset.NodeId()] {
		if s.WaitFor != "" {
			for {
				if <-displayCh == s.WaitFor {
					break
				}
			}
		}
		for _, i := range s.Issues {
			fmt.Println(i)
			outChan <- i
			chat.Post(g, i)
		}
	}

	for {
		select {
		case <-displayCh:
			// do nothing
		case <-doneChan:
			causallyOrdered := true

			// The messages must be unique!
		testLoop:
			for _, writeSet := range script {
				for _, writeStep := range writeSet {
					before := writeStep.WaitFor
					if before == "" {
						continue
					}
					after := writeStep.Issues

					beforeIndex := slices.Index(outputs, before)
					for _, msg := range after {
						// make sure that in output, issue comes after before
						msgIndex := slices.Index(outputs, msg)
						if msgIndex < beforeIndex {
							causallyOrdered = false
							fmt.Println(red + "BAD: '" + msg + "' CAME BEFORE '" + before + "'" + reset)
							break testLoop
						}
					}
				}
			}

			if causallyOrdered {
				fmt.Println(green + "✅✅✅✅✅✅ You are cracked. The messages are causally ordered. go touch some grass ✅✅✅✅✅✅" + reset)
			} else {
				fmt.Println(red + "❌❌❌❌❌❌ Not causally ordered... 😞 not cracked unfortunately ❌❌❌❌❌❌" + reset)
			}

		}
	}
}

type step struct {
	WaitFor string
	Issues  []string
}
type channel struct {
	from uint32
	to   uint32
}

var script = make(map[uint32][]step)
var slowChannels []channel
var numNodes int

func ws(nodeId uint32, waitFor string, issue ...string) {
	script[nodeId] = append(script[nodeId], step{waitFor, issue})
}

var outputs []string

const (
	red   = "\033[31;1m" // Bold Red
	green = "\033[32;1m" // Bold Green
	reset = "\033[0m"    // Reset color
)
