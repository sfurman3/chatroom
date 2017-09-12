// Server is an implementation of a distributed, FIFO consistent chatroom where
// participants (servers) can broadcast messages and detect failures. Each
// server keeps a FIFO log of messages it has received.
//
// "server [id] [numservers] [port]" sets up a server with ID [id] on port
// [20000 + id] with a master-facing port of [port] (i.e the port which
// the master process uses to issue commands and accept responses).
// [numservers] is the total number of servers in the system, and is used to
// connect to the remaining servers. A system of n servers is assumed to have
// server IDs {0...n-1} and ports {20000...20000 + n-1} respectively.
//
//  The following master commands are supported:
//  --------------------------------------------
//  - "get\n:               return a list of all received messages
//  - "alive\n":            return a list of server IDs believed to be alive
//  - "broadcast <m>\n":    send <m> to everyone alive (including the sender)
//
//  Responses have the following format:
//  ------------------------------------
//  - "get\n"   -> "messages <msg1>,<msg2>,...\n"
//  - "alive\n" -> "alive <id1>,<id2>,...\n"
//
// You can test a server instance using netcat. For example:
//  ➜  server 0 1 30000 &
//  [2] 43246
//  ➜  netcat localhost 30000
//  get                         (command)
//  messages
//  alive                       (command)
//  alive 0
//  broadcast hello world       (command)
//  get                         (command)
//  messages hello world
//  ^C
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
	HEARTBEAT_INTERVAL = 200 * time.Millisecond

        // Maximum interval after the send timestamp of the last message
        // received from a server for which the sender is considered alive
	ALIVE_INTERVAL = 250 * time.Millisecond

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

///////////////////////////////////////////////////////////////////////////////
// server                                                                    //
///////////////////////////////////////////////////////////////////////////////

func main() {
	// Bind the master-facing and server-facing ports and start listening
	go serveMaster()
	go fetchMessages()
	heartbeat()
}

// heartbeat sleeps for HEARTBEAT_INTERVAL and broadcasts an empty message to
// every server to indicate that the server is still alive
func heartbeat() {
	for {
		time.Sleep(HEARTBEAT_INTERVAL)
		go broadcast(emptyMessage())
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

// serveMaster listens on MASTER_PORT for a connection from a master process
// and services its commands
//
// NOTE: only one master process is served at any given time
func serveMaster() {
	// Bind the master-facing port and start listen for commands
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(MASTER_PORT))
	if err != nil {
		Fatal("failed to bind master-facing port: ",
			strconv.Itoa(MASTER_PORT))
	}

	for {
		masterConn, err := ln.Accept()
		if err != nil {
			Fatal(err)
		}

		handleMaster(masterConn)
	}
}

// handleMaster executes commands from the master process and responds with any
// requested data
func handleMaster(masterConn net.Conn) {
	defer masterConn.Close()

	master := bufio.NewReadWriter(
		bufio.NewReader(masterConn),
		bufio.NewWriter(masterConn))

	for {
		command, err := master.ReadString('\n')
		if err != nil {
			// connection to master lost
			return
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
				for id := 0; id < ID; id++ {
					// add all server ids for which a
					// heartbeat was sent within the
					// alive interval
					if now.Sub(stmps[id]) < ALIVE_INTERVAL {
						master.WriteString(strconv.Itoa(id))
						master.WriteByte(',')
					}
				}
				master.WriteString(strconv.Itoa(ID))
				for id := ID + 1; id < NUM_PROCS; id++ {
					// add all server ids for which a
					// heartbeat was sent within the
					// heartbeat interval
					if now.Sub(stmps[id]) < HEARTBEAT_INTERVAL {
						master.WriteByte(',')
						master.WriteString(strconv.Itoa(id))
					}
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
