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

package main

import (
	"bufio"
	"bytes"
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

// Message represents a message sent from one server to another
type Message struct {
	Id      int       `json:"id"`  // server id
	Rts     time.Time `json:"rts"` // real-time timestamp
	Content string    `json:"msg"` // content of the message
}

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

func init() {
	flag.IntVar(&ID, "id", ID, "id of the server {0, ..., n-1}")
	flag.IntVar(&NUM_PROCS, "n", NUM_PROCS, "total number of servers")
	flag.IntVar(&MASTER_PORT, "port", MASTER_PORT, "number of the "+
		"master-facing port")
	flag.Parse()

	setArgsPositional()

	PORT = START_PORT + ID
	LastTimestamp.value = make([]time.Time, NUM_PROCS)
}

// TODO: MOVE?
// TODO:NOTE: IDs used by data structures and functions in the vector package are
// always the server ID plus 1.

// Error logs the given error
func Error(err ...interface{}) {
	log.Println(ERROR, err)
}

// Fail logs the given error and exits with status 1
func Fatal(err ...interface{}) {
	log.Fatalln(ERROR, err)
}

// TODO: YOU NEED THREAD SAFE CODE
// TODO: TURN OFF LOGGING
func main() {
	// TODO: Receipt times can be stored in a map from messages to timestamps
	// TODO: In order to maintain FIFO ordering, store each received
	// message in a FIFO slice log of messages
	// TODO: Maintain a message receptacle of received messages; for every
	// receive: add it to the receptacle, if successful store it in the
	// FIFO slice

	// TODO: heartbeat should be less than 1 second, shouldn't be too long
	// Too short and congestion
	// Too long and you don't know for a while whether or not the server is
	// alive

	// --------------------------------------------------
	// TODO: Synchronize servers
	// TODO: Synchronize message delivery
	go serveMaster()
	go fetchMessages()
	heartbeat() // TODO
}

func heartbeat() {
	for {
		time.Sleep(HEARTBEAT_INTERVAL)
		go broadcast(emptyMessage())
	}
}

func emptyMessage() *Message {
	return &Message{
		Id:  ID,
		Rts: time.Now(),
	}
}

func newMessage(msg string) *Message {
	return &Message{
		Id:      ID,
		Rts:     time.Now(),
		Content: msg,
	}
}

func fetchMessages() {
	// Bind the server-facing port and listen for messages
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(PORT))
	if err != nil {
		Fatal("failed to bind server-facing port:", strconv.Itoa(PORT))
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go handleMessenger(conn)
	}
}

// TODO
func handleMessenger(conn net.Conn) {
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

// TODO
func serveMaster() {
	// Bind the master-facing port and start listen for commands
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(MASTER_PORT))
	if err != nil {
		Fatal("failed to bind master-facing port:",
			strconv.Itoa(MASTER_PORT))
	}

	masterConn, err := ln.Accept()
	if err != nil {
		Fatal(err)
	}
	defer masterConn.Close()

	buff := bytes.NewBuffer(make([]byte, NUM_PROCS))
	rdr := bufio.NewReader(masterConn)
	wrtr := bufio.NewWriter(masterConn)
	master := bufio.NewReadWriter(rdr, wrtr)

	for {
		command, err := master.ReadString('\n')
		if err != nil {
			Fatal("master may have been terminated")
		}

		command = strings.TrimSpace(command)
		switch command {
		case "get":
			MessagesFIFO.mutex.Lock()
			for _, msg := range MessagesFIFO.value {
				buff.WriteString(msg.Content)
				buff.WriteByte(',')
			}
			MessagesFIFO.mutex.Unlock()
			b := buff.Bytes()
			if len(b) != 0 && b[len(b)-1] == ',' {
				b = b[:len(b)-1]
			}
			buff.Reset()

			master.WriteString("messages ")
			master.Write(b)
			master.WriteByte('\n')
			err = master.Flush()
			if err != nil {
				Fatal(err)
			}
		case "alive":
			now := time.Now()
			LastTimestamp.mutex.Lock()
			for id, ts := range LastTimestamp.value {
				if now.Sub(ts) < HEARTBEAT_INTERVAL || id == ID {
					buff.WriteString(strconv.Itoa(id))
					buff.WriteByte(',')
				}
			}
			LastTimestamp.mutex.Unlock()
			b := buff.Bytes()
			if len(b) != 0 && b[len(b)-1] == ',' {
				b = b[:len(b)-1]
			}
			buff.Reset()

			master.WriteString("alive ")
			master.Write(b)
			master.WriteByte('\n')
			err = master.Flush()
			if err != nil {
				Fatal(err)
			}
		default:
			broadcastComm := "broadcast "
			if !strings.HasPrefix(command, broadcastComm) {
				Error(fmt.Errorf(
					"unrecognized command: \"%s\"",
					command))
				continue
			}

			message := command[len(broadcastComm):]
			broadcast(newMessage(message))
		}
	}
}

// TODO
// NOTE: Does not require synchronization
func broadcast(msg *Message) {
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

		conn, err := net.Dial("tcp", ":"+strconv.Itoa(START_PORT+id))
		if err != nil {
			continue
		}
		defer conn.Close()

		wrtr := bufio.NewWriter(conn)
		msgBytes, err := json.Marshal(msg)
		_, err = wrtr.Write(msgBytes)
		if err != nil {
			continue
		}

		err = wrtr.WriteByte('\n')
		if err != nil {
			continue
		}

		_ = wrtr.Flush()
	}
}

// If no arguments were provided via flags, parse the first three arguments
// into ID, NUM_PROCS, and PORT respectively
func setArgsPositional() {
	getIntArg := func(i int) int {
		arg := flag.Arg(i)
		if arg == "" {
			fmt.Fprintf(os.Stderr, "%v: missing one or more "+
				"arguments (there are %d)\n",
				os.Args, len(REQUIRED_ARGUMENTS))
			flag.PrintDefaults()
			os.Exit(1)
		}
		val, err := strconv.Atoi(arg)
		if err != nil {
			log.Fatal("could not parse: '%v' into an integer\n", arg)
		}
		return val
	}

	for idx, val := range REQUIRED_ARGUMENTS {
		if *val == -1 {
			*val = getIntArg(idx)
		}
	}
}
