## Run Instructions
There's a small message client built on top of the RPC service. To run it:

1. Start the db daemon:
   `go run cmd/dbd/dbd.go &`
   - It will start listening on an address/port and print out that address.
2. Start the message daemon:
   `go run cmd/messaged/messaged.go addr:port &`
   - where `addr:port` is the address/port the db daemon is listening on.
3. Start the auth daemon:
   `go run cmd/authd/authd.go addr:port &`
4. Finally, connect as many chat clients as you would like.
   `go run cmd/chat/chat.go 127.0.0.1:46756 s/l user pass`
   -  `s` will sign a new user up, `l` will allow you to login to an existing user account.
   - To send a message, use:
        - `s <user> <message>`
   - To read your inbox, use:
        -  `read`
   -  To allow another user to send you messages, use:
        - `allow <user>`
   - To block another user, use:
        - `block <user>`
   - **NEW** Push/Pull messaging using causal multicast:
      - To enable push messaging (messages will be delivered without the need to 'pull'), use:
          - `push`
      - To go back ot pull messaging, use:
          - `pull`

### Test Instructions
- There is an end-to-end test suite implemented, that will simulate chats between clients, and assert that all messages are delivered in causal order. To use, run:
  - `./cmd/test/runTestCaseGrid.sh numWindows testName`
        - `testName` is one of the json files (without the `.json` suffix) in the `testInputs` folder.
        - `numWindows` is the number of clients spawned. Each client will be spawned in a separate terminal window, the windows will be placed in a grid.
  - Please note that this test script only works on macos with iTerm2 installed. It relies on appscript to spawn and organize the terminal windows.
  - You can manually run a test by starting all the daemons, and then launching the desired number of clients manually with:
        - `go run cmd/testChat/testChat.go addr:port testName`

Clearly and concisely addresses at least the following points.
1. If you used our implementation for `transport` and `rpc`, describe briefly the strengths and weakness of the `transport` protocol.  Compare our `rpc` package to yours.
    - **Strengths**
      - Implicit acks: The provided package waits briefly before sending an ACK; if the call completes and a reply is sent, the ack is skipped, reducing unnecessary messages. 
      - Effective resource management: The provided package uses resources very effectively. It cleans up pending calls and channels, along with a simple system to remove obsolete hosts.
    - **Weaknesses**
      - Complexity: Implementing reliability on top of UDP adds significant complexity to the transport package, making debugging slightly more challenging. A TCP based solution would help to simplify the design, and guarantee that the reliability works without any bugs, albeit with slightly more overhead.
      - One socket for all clients: In the provided implementation, one socket is used for all listening. This technically limits the ability to concurrently receive messages, although the calls are still processed asynchronously.  A TCP implementation would use a separate socket + thread for each client, allowing the recipt of messages to happen concurrently.
    - **Comparison**
      - The provided package is more modular. It utilizes a Golang feature that we were unaware of during L1 -- generics. This allows client/server stubs to share much more code, making extension with new stubs easier and simpler.
      - Both packages provide a similar level of transparancy. They both abstract away transport details from the application so it may call the RPC stubs as if they were local functions executing in the same program.
 2. If you used your own implementation based on Lab 1, briefly describe the changes you had to make to complete Lab 2.
    - N/A 
 3. If you used a standard rpc module, briefly describe its API.
    - N/A 
 4. Evaluate how you added timestamps to messages from the perspective of transparency and modularity.  How similar/differ are normal rpc calls from multicast calls?
    - **Transparency:** 
      - Non-multicast RPC calls bypass timestamp processing completely, but pass through the corresponding handler functions. The system inspects the packets to determine if they are a multicast call, and then injects/extracts clock info into the payload. It only delivers messages to the application when the vector clock deems them causally deliverable, maintaining transparancy.
      - In terms of transparency -- the RPC layer (outside the multicast packages) and the application layer do not care about the timestamps or clocks at all. They behave as if the multicast calls were normal RPC calls delivered by the network in the correct causal order.
    - **Modularity:** 
      - Modularity is *slightly* rougher. The code assumes that "Delivery" is the only multicast function and only supports one argument. Supporting additional functions and signatures would require some minor refactoring. Namely, the `peekFuncName` function would need some abstraction and the `Multicast` and `ScheduleDelivery` would need a list of all the multicast function names.
 5. Evaluate the correctness of your implementation?  Does it order messages correctly?  Does it ONLY order messages that are causally ordered?
    - **Causal Ordering**:
      - We are confident that our implementation respects causal ordering. This is because it will only deliver a message to the application under two conditions:
        1. At the index of the sending node, the incoming timestamp is exactly one more than our local timestamp.
        2. Every other time value contained within the incoming timestamp is less than or equal to the corresponding time value of the local timestamp (i.e. `incomingTimestamp[k] <= localTimestamp[k]` for all `k` != the node id of the sender process`) 
      - The first condition ensures that all earlier messages from the sender have been delivered. This respects the happened-before relation from the perspective of the sending client -- all messages that particular client sent before the current message must have happened before the current message was sent.
      - The second condition ensures that all messages from other nodes that causally affect the current message have also been delivered. This respects the happened-before relation from the perspective of the other nodes -- all messages sent by other clients that happened-before the client sent its current message must be delivered before the client's current message.
    - **Not enforcing stricter order**:
      - We are also confident that we do not enforce an ordering stricter than causal ordering. The two conditions above do not impose constraints beyond checking causality. If two messages are concurrent and there is no causal relation between them (the two clocks have mix of smaller/larger time values relative to each other) the algorithm does not enforce an order on their delivery. When running tests with multiple messages that all depend on the same message to happen-before (concurrent), the messages are regularly delivered in arbitrary orders.
    - **Leaving and Joining**:
      - Departed client's clock values are kept in the clock vector, and their ids are retained in a permanent address-to-ID map, so even if a message gets delivered by the network after a client has left the nodeset, other nodes will still know what their id was and can check the clock to tell if the message should be delivered. This is slightly wasteful, but helps ensure correctness.
      - We make no attempt to update a newly joined client with previous messages, so any messages that causally depend on messages before they joined will not be delivered.
 6. Evaluate your testing strategy.  How confident are you that your solution will behave correctly under all inputs?  Why are you confident or not?
    - We are fairly confident that our solution will behave correctly under all inputs due to the thoroughness of our test suite. To test our implementation, we created an automated end-to-end test suite (`cmd/testChat/testChat.go`) which parses all node write sets and slow channels from a JSON file. We also wrote a bash script to startup the server process (`nodesetd`) and all client node processes required to trigger message sending (this was mainly to streamline the testing process and make it easier for us to repetitively run our tests when debugging). Once all of these processes are spawned, the client nodes begin multicasting messages using the same logic as `scriptedChat.go`. The key component of our test suite is that once a client receives a message, in addition to displaying it, the client stores it in a message output list. Once all messages are received, the output list is checked for valid causality based on the test write sets which determines if the test case fails or not.
    - We test the following scenarios: 
      - Simple causal dependencies (eg. a -> b -> c)
      - Branching causal dependencies (eg. a -> b and a -> c on separate nodes)
      - Independent parallel chains of causal dependencies (eg. a -> b -> c and d -> e -> f) 
      - Large number of clients (16) using simple causal dependencies
      - Alternating between fast and slow channels (a (slowly) -> b -> c (slowly) -> d)
    - If you are curious about the specific tests, please see `/cmd/testChat/testInputs` for the specific cases. They are written in a JSON format that mirrors the inputs to `ws`. You can run the tests by calling `sh ./cmd/testChat/runTestCaseGrid.sh numNodes testcase` where `numNodes` is the number of nodes in the test case and `testcase` is the name of the testcase without the `.json` suffix. (This only works on macs with iTerm2 installed). Or, just manually open the required number of `testChat.go` windows with the ip and the testcase name.
    - An improvement would be to have unit tests for the vector clock logic as well. Our end-to-end tests provide us with confidence that our test suite works as a whole, but it makes debugging more difficult, and there is a chance that bugs elsewhere in the system could mask errors in the clock logic, giving us the false impression that the code is working correctly.
    - We tested leaving/joining manually, another improvement would be to add leaving/joining clients into the automated test suite.
### Other notes:
- Limitations:
- **Performance**:
  - Our `scheduleDelivery` polls to check if the message is deliverable. This is a bit of a waste of resources, and the sleep could introduce latency that is not needed. Switching to a condition variable could reduce resource wastefulness and latency.
- **Dying before a message is sent**
  - The system handles the case where a client dies after all messages are sent. However, if a client dies during a slow-channel delay, the message might not get sent and our system does not recover from this case.
