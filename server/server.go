// TODO: Server implementation

// TODO: Cite the paper: "Consistent Global States of Distributed Systems": by
// Özalp Babaoğlu and Keith Marzullo.

// TODO: REORGANIZE
// Concepts & Terminology:
// -----------------------
// - "distributed system"
//   + a collection of sequential 'processes' p_1, ..., p_n and a network
//     capable of implementing unidirectional communication 'channels' between
//     pairs of processes for message exchange. Channels are reliable but may
//     deliver messages out of order.
//   + p_0 denotes a monitor process, which observes the system in order to
//     construct global states
//
// - "events": we define 3 possible events
//    1. local events
//    2. send(m)         (send events)
//    3. receive(m)      (receive events)
//
//    NOTE: monitoring does not generate any new events
//
//   + e_i^n ::= the n'th event of process p_i
//
// - "causal precedence"
//   + e -> e'    (e causally precedes e' (i.e. "happens before"))
//   + causal precedence implies that event e MAY HAVE influenced event e'
//
// - "concurrency"
//   + e || e' <=> NOT(e -> e') AND NOT(e' -> e)    (e and e' are concurrent)
//   + by definition concurrent events ARE NOT causally related (i.e. neither
//     influenced the other)
//
// - "local history"
//   + a (possibly infinite) sequence h_i of events local to a process p_i
//
// - "global history"
//   + a set H ::= h_1 (union) h_2 ... (unordered) containing all events in a
//     system
//
// - "global state"
//   + an n-tuple of states (sigma_i, sigma_j, ..., sigma_n) one for each
//     process
//
// - "cut"
//   + a subset C of a system's global history H, containing the union of the
//     first x events in each process, where x may be different for each
//     process
//   + the set containing the most recent event in each process is called the
//     "frontier" of the cut, which has a corresponding "global state"
//
// - "run"
//   + a total ordering R that contains all the events in the global history
//     and is 'consistent' with each local history (i.e. the events of a
//     process in the run occur in the same relative order as they do in their
//     local history)
//
// - "consistency"
//   + multiple objects defined here can be consistent:
//     * cut: (e in C) AND (e' -> e) => e' in C
//       - i.e. for every event in the cut, there is also every event that
//         causally precedes it
//	 - simply, for every receive in the cut, there is also a corresponding
//	   send
//	 - each consistent cut corresponds to a "consistent global state"
//     * NOTE: a run corresponds to a sequence of global states. if a run is
//       consistent, then so are the global states in the sequence
//
// - "receive"
//   + the 'event' of a message arriving at its destination (but before it is
//     passed to the recipient process)
//
// TODO: IMPORTANT
// - "observation" (of a distributed computation by a "monitor" process)
//   + a permutation of a subset of a run, which can thus be consistent OR
//     inconsistent
//   + a "consistent observation" MUST correspond to a consistent run
//   + observations are constructed by monitors from events received from
//     processes in a system
//   + NOTE: we can use consistent observations to construct consistent global
//     states
//     * because consistent observations correspond to consistent runs,
//       consistent runs can be used to construct consistent cuts, and
//       consistent cuts have frontiers that correspond to consistent global
//       states
//
// - "deliver"
//   + the _act_ of presenting received messages to a process
//   + a "delivery rule" is a rule for when to deliver received messages
//
// - "causal delivery"
//   + send_i(m) -> send_j(m) => receive_i(m) -> receive_j(m)
//   + the property of delivering messages in an order that preserves causal
//     precedence (i.e. that yields consistent observations)
//
// - "clock condition":
//   + e' -> e => C(e') -> C(e)    (C(e) is the clock timestamp of event e)
//
// - "logical clocks"
//   + a local variable that maps events to the positive natural numbers
//   + satisfy the "clock condition" (above)
//   + each process initializes LC to 0
//   + each sent message m contains a timestamp TS(m) == LC(send(m))
//   + update rules:
//     * LC(e_i) := {
//          LC + 1,                 (if e_i is a local or send event send(m))
//          max{LC, TS(m)} + 1      (if e_i == receive(m))
//       }
//
// - "gap-detection"
//   + the ability to determine, given two events e and e' with clock
//     values LC(e) < LC(e'), whether there exists some e'' such that
//     LC(e) < LC(e'') < LC(e')
//
// - "stable"
//   + a message m received by process p is stable if no future messages
//     with timestamps smaller than TS(m) can be received by p.
//   + NOTE: because we assume channels are reliable, stability of a message m
//     at p_0 can be guaranteed when p_0 has received at least one message from
//     ALL other processes with a timestamp greater than TS(m)
//     * because some processes may not communicate p_0 after a certain point,
//       stability of a message can be guaranteed by requesting an
//       acknowledgement from all processes to a periodic empty message, which
//       “flushes out” messages that may have been in the channels.
//
// TODO
// - "strong clock condition"
//   + e -> e' <=> TC(e) < TC(e')

// - "vector clock"
// - "hidden channels"
// - "snapshot protocols"
// - "properties of snapshots"
// - "global predicates"
// - "stable predicates"
// - "nonstable predicates"
// - "multiple monitors"

// TODO:NOTE: IDs used by data structures and functions in the vector package are
// always the server ID plus 1.

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Base port for servers in the system
	// Port numbers are always START_PORT + ID
	START_PORT = 20000

	// Duration between heartbeat messages (i.e. empty messages broadcasted
	// to other servers to indicate the server is alive)
	HEARTBEAT_INTERVAL = 500 * time.Millisecond

	// Constants for printing error messages to the terminal
	BOLD_RED = "\033[31;1m"
	NO_STYLE = "\033[0m"
	ERROR    = "[" + BOLD_RED + "ERROR" + NO_STYLE + "]"
)

var (
	ID                 = -1 // id of the server {0, ..., NUM_PROCS-1}
	NUM_PROCS          = -1 // total number of servers
	MASTER_PORT        = -1 // number of the master-facing port
	REQUIRED_ARGUMENTS = []*int{&ID, &NUM_PROCS, &MASTER_PORT}

	PORT = -1 // server's port number

	// struct containing all received messages in FIFO order
	MessagesFIFO struct {
		value []*Message
		mutex sync.Mutex // mutex for accessing contents
	}

	// struct containing the timestamp of the last message from each server
	LastTimestamp struct {
		value []time.Time
		mutex sync.Mutex // mutex for accessing contents
	}
)

// Message represents a message sent from one server to another
type Message struct {
	Id      int       `json:"id"`  // server id
	Rts     time.Time `json:"rts"` // real-time timestamp
	Content string    `json:"msg"` // content of the message
}

// emptyMessage returns an empty message with a timestamp of time.Now()
func emptyMessage() *Message {
	return &Message{
		Id:  ID,
		Rts: time.Now(),
	}
}

// newMessage returns a message with Content msg and a timestamp of time.Now()
func newMessage(msg string) *Message {
	return &Message{
		Id:      ID,
		Rts:     time.Now(),
		Content: msg,
	}
}

// init parses and validates command line arguments (by name or position) and
// initializes global variables
func init() {
	flag.IntVar(&ID, "id", ID, "id of the server {0, ..., n-1}")
	flag.IntVar(&NUM_PROCS, "n", NUM_PROCS, "total number of servers")
	flag.IntVar(&MASTER_PORT, "port", MASTER_PORT, "number of the "+
		"master-facing port")
	flag.Parse()

	setArgsPositional()

	if NUM_PROCS <= 0 {
		Fatal("invalid number of servers: ", NUM_PROCS)
	}

	PORT = START_PORT + ID
	LastTimestamp.value = make([]time.Time, NUM_PROCS)
}

// setArgsPositional parses the first three command line arguments into ID,
// NUM_PROCS, and PORT respectively. It should be called if no arguments were
// provided via flags.
func setArgsPositional() {
	getIntArg := func(i int) int {
		arg := flag.Arg(i)
		if arg == "" {
			fmt.Fprintf(os.Stderr, "%v: missing one or more "+
				"arguments (there are %d)\n"+
				"(e.g. \"%v 0 1 10000\" OR \"%v -id 0 -n 1 "+
				"-port 10000)\"\n\n",
				os.Args, len(REQUIRED_ARGUMENTS),
				os.Args[0], os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
		val, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"could not parse: '%v' into an integer\n", arg)
		}
		return val
	}

	for idx, val := range REQUIRED_ARGUMENTS {
		if *val == -1 {
			*val = getIntArg(idx)
		}
	}
}

// Error logs the given error
func Error(err ...interface{}) {
	log.Println(ERROR + " " + fmt.Sprint(err...))
}

// Fail logs the given error and exits with status 1
func Fatal(err ...interface{}) {
	log.Fatalln(ERROR + " " + fmt.Sprint(err...))
}

// TODO: Receipt times can be stored in a map from messages to timestamps
// TODO: In order to maintain FIFO ordering, store each received
// message in a FIFO slice log of messages
// TODO: Maintain a message receptacle of received messages; for every
// receive: add it to the receptacle, if successful store it in the
// FIFO slice

// TODO: Explain command line arguments: type -h for more information
// TODO: message are delimited by '\n'
// TODO: Give an example with netcat (in a test file?)
// TODO: Whether a server is alive or not is determined by regularly sending a
// heartbeat (i.e. a timestamped empty message) after a period of
// HEARTBEAT_INTERVAL. Recipient processes use the timestamp from the last
// received message. If it was sent after (now - HEARTBEAT_INTERVAL), then the
// server is reported alive.
func main() {
	// Bind the master-facing and server-facing ports and start listening
	go serveMaster()
	go fetchMessages()

	// Sleep for a bit to let other servers set up server-facing ports
	// before delivering the first heartbeat
	time.Sleep(100 * time.Millisecond)
	heartbeat()
}

// heartbeat sleeps for HEARTBEAT_INTERVAL and broadcasts an empty message to
// every server to indicate that the server is still alive
func heartbeat() {
	for {
		go broadcast(emptyMessage())
		time.Sleep(HEARTBEAT_INTERVAL)
	}
}

// fetchMessages retrieves messages from other servers and adds them to the
// log, listening on PORT (i.e. START_PORT + PORT)
func fetchMessages() {
	// Bind the server-facing port and listen for messages
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(PORT))
	if err != nil {
		Fatal("failed to bind server-facing port: ", strconv.Itoa(PORT))
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		handleMessage(conn)
	}
}

// handleMessage retrieves the first message from conn, adds it to the log, and
// closes the connection. It also updates LastTimestamp for the sending server.
//
// NOTE: This function must be called sequentially (NOT by starting a new
// thread for each new connection) in order to maintain FIFO receipt.
// Otherwise, depending on scheduling, a message B may be added to MessagesFIFO
// before another message A, even though A connected first.
//
// The disadvantage is that, if the delivery of a message is blocked (e.g. the
// sender died before it could terminate the message with a '\n'), then all of
// the subsequent messages to be delivered are also blocked, possibly FOREVER.
//
// NOTE: If FIFO receipt is no longer necessary, we can simply sort
// MessagesFIFO by send timestamp in order to approximate the send order. We
// could also use a causal delivery method provided by a data structure such as
// the vector.MessageReceptacle to deliver messages based on causal precedence.
func handleMessage(conn net.Conn) {
	defer conn.Close()

	messenger := bufio.NewReader(conn)
	msg := new(Message)
	msgBytes, err := messenger.ReadBytes('\n')
	if err != nil {
		return
	}

	err = json.Unmarshal(msgBytes, msg)
	if err != nil {
		return
	}

	// Update the heartbeat metadata
	// NOTE: assumes message IDs are in {0..n-1}
	LastTimestamp.mutex.Lock()
	LastTimestamp.value[msg.Id] = msg.Rts
	LastTimestamp.mutex.Unlock()

	if len(msg.Content) == 0 { // msg is an empty message
		return
	}

	MessagesFIFO.mutex.Lock()
	MessagesFIFO.value = append(MessagesFIFO.value, msg)
	MessagesFIFO.mutex.Unlock()
}

// serveMaster executes commands from the master process (listening on
// MASTER_PORT) and returns any requested data
func serveMaster() {
	// Bind the master-facing port and start listen for commands
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(MASTER_PORT))
	if err != nil {
		Fatal("failed to bind master-facing port: ",
			strconv.Itoa(MASTER_PORT))
	}

	masterConn, err := ln.Accept()
	if err != nil {
		Fatal(err)
	}
	defer masterConn.Close()

	master := bufio.NewReadWriter(
		bufio.NewReader(masterConn),
		bufio.NewWriter(masterConn))

	for {
		command, err := master.ReadString('\n')
		if err != nil {
			Fatal("master may have been terminated")
		}

		command = strings.TrimSpace(command)
		switch command {
		case "get":
			master.WriteString("messages ")
			MessagesFIFO.mutex.Lock()
			if len(MessagesFIFO.value) > 0 {
				msgs := MessagesFIFO.value
				lst := len(msgs) - 1
				for _, msg := range msgs[:lst] {
					master.WriteString(msg.Content)
					master.WriteByte(',')
				}
				master.WriteString(msgs[lst].Content)
			}
			MessagesFIFO.mutex.Unlock()
			master.WriteByte('\n')

			err = master.Flush()
			if err != nil {
				Fatal(err)
			}
		case "alive":
			now := time.Now()

			master.WriteString("alive ")
			LastTimestamp.mutex.Lock()
			{
				stmps := LastTimestamp.value
				lst := len(stmps) - 1
				for id, ts := range stmps[:lst] {
					// add all server ids for which a
					// heartbeat was sent within the
					// heartbeat interval
					if now.Sub(ts) < HEARTBEAT_INTERVAL ||
						id == ID {
						master.WriteString(strconv.Itoa(id))
						master.WriteByte(',')
					}
				}
				if now.Sub(stmps[lst]) < HEARTBEAT_INTERVAL ||
					lst == ID {
					master.WriteString(strconv.Itoa(lst))
				}
			}
			LastTimestamp.mutex.Unlock()
			master.WriteByte('\n')

			err = master.Flush()
			if err != nil {
				Fatal(err)
			}
		default:
			broadcastComm := "broadcast "
			if !strings.HasPrefix(command, broadcastComm) {
				Error("unrecognized command: \"", command, "\"")
				continue
			}

			message := command[len(broadcastComm):]
			broadcast(newMessage(message))
		}
	}
}

// broadcast sends the given message to all other servers (including itself and
// excluding the master)
//
// NOTE: Sends are sequential, so that broadcast does not return until an
// attempt has been made to send the message to all servers
//
// NOTE: This function must be called sequentially (NOT by starting a new
// thread for each new message) in order to maintain FIFO receipt. Otherwise,
// depending on scheduling, a message B could be broadcast to a server before
// another message A, even though A's thread was started first.
//
// The disadvantage is that, if the receipt of one message is delayed for any
// of its recipients, then all of the subsequent commands sent by the master
// are also delayed (until the send times out). This may cause servers to not
// receive the message on time. This is likely not an issue when working with a
// small number of servers.
//
// NOTE: If FIFO receipt is no longer necessary, the recipient can simply sort
// delivered messages by send timestamp in order to approximate the send order.
// They could also use a causal delivery method provided by a data structure
// such as the vector.MessageReceptacle to deliver messages based on causal
// precedence.
func broadcast(msg *Message) {
	// Convert to JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}
	msgJSON := string(msgBytes)

	// send non-empty messages to self
	if len(msg.Content) != 0 {
		MessagesFIFO.mutex.Lock()
		MessagesFIFO.value = append(MessagesFIFO.value, msg)
		MessagesFIFO.mutex.Unlock()
	}

	// send message to other servers
	for id := 0; id < NUM_PROCS; id++ {
		if id == ID {
			id++
		}

		// NOTE: In the future, you may want to consider using
		// net.DialTimeout (e.g. the recipient is so busy it cannot
		// service the send in a reasonable amount of time) and/or
		// consider starting a new thread for every send to prevent
		// sends from blocking each other (the timeout might help
		// prevent a buildup of threads that can't progress)
		conn, err := net.Dial("tcp", ":"+strconv.Itoa(START_PORT+id))
		if err != nil {
			continue
		}
		defer conn.Close()

		fmt.Fprintln(conn, msgJSON)
	}
}
