package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sfurman3/chatroom/logical"
)

func TestNewMessage(t *testing.T) {
	clk, _ := NewClockBuilder().Id(1).Length(1).Build()
	b, err := json.Marshal(NewMessage("hi, hello", clk))
	if err != nil {
		t.Fail()
	}

	expected := `{"msg":"hi, hello","ts":{"id":1,"v":["0"]}}`
	if string(b) != expected {
		t.Fatalf("expected: %s, got: %s", expected, string(b))
	}
}

func TestClock_String(t *testing.T) {
	var clk *Clock
	if clk.String() != "<nil>" {
		fmt.Println(clk.String(), "!= \"<nil>\"")
		t.Fail()
	}

	var clkVal Clock
	if clkVal.String() != "[]" {
		fmt.Println(clkVal.String(), "!= \"[]\"")
		t.Fail()
	}

	clk, _ = NewClockBuilder().Length(1).Id(1).Build()
	if clk.String() != "[0]" {
		fmt.Println(clk.String(), "!= \"[0]\"")
		t.Fail()
	}

	clk, _ = NewClockBuilder().Length(2).Id(1).Build()
	if clk.String() != "[0, 0]" {
		fmt.Println(clk.String(), "!= \"[0, 0]\"")
		t.Fail()
	}
}

func TestClock_Equal(t *testing.T) {
	clk, other := new(Clock), new(Clock)
	if !clk.Equal(other) {
		t.Fatal("shouldn't fail for uninitialized clocks")
	}

	clk, _ = NewClockBuilder().Id(1).Length(3).Build()
	if clk.Equal(other) {
		t.Fatal("should fail for clocks of unequal length")
	}

	other, _ = NewClockBuilder().Id(2).Length(3).Build()
	if !clk.Equal(other) {
		t.Fatal("shouldn't fail for clocks with different IDs")
	}
}

func TestClock_ErrComparableTo(t *testing.T) {
	clk, other := new(Clock), new(Clock)
	if clk.ErrComparableTo(other) == nil {
		t.Fatal("should fail for uninitialized clocks")
	}

	clk, _ = NewClockBuilder().Id(1).Length(3).Build()
	if clk.ErrComparableTo(other) == nil {
		t.Fatal("should fail for clocks of unequal length")
	}

	other, _ = NewClockBuilder().Id(2).Length(3).Build()
	if clk.ErrComparableTo(other) != nil {
		t.Fatal("shouldn't fail for clocks with different IDs")
	}
}

func TestClock_LessThan(t *testing.T) {
	clk, _ := NewClockBuilder().Id(1).Length(3).Build()
	other, _ := NewClockBuilder().Id(1).Length(3).Build()
	if clk.LessThan(other) {
		t.Fatal("should be false for uninitialized threads")
	}
	other.TickLocal()
	if !clk.LessThan(other) {
		t.Fatalf("%s < %s should be true", clk, other)
	}
	if other.LessThan(clk) {
		t.Fatalf("%s < %s should be false", clk, other)
	}

	clk, _ = NewClockBuilder().Id(1).Length(3).Build()
	other, _ = NewClockBuilder().Id(2).Length(3).Build()
	other.TickLocal()
	if !clk.LessThan(other) {
		t.Fatalf("%s < %s should be true", clk, other)
	}
	if other.LessThan(clk) {
		t.Fatalf("%s < %s should be false", clk, other)
	}

	clk, _ = NewClockBuilder().Id(1).Length(3).Build()
	other, _ = NewClockBuilder().Id(1).Length(3).Build()
	if clk.LessThan(other) {
		t.Fatalf("%s < %s should be false", clk, other)
	}
}

func ExampleClock_UnmarshalJSON() {
	vecClockJsonBytes := []byte(`{"id":1,"v":["1","0","0","0","0"]}`)

	vecClock := new(Clock)
	_ = json.Unmarshal(vecClockJsonBytes, vecClock)
	fmt.Println("vecClock:", vecClock)
	// Output:
	// vecClock: [1, 0, 0, 0, 0]
}

func ExampleClock_MarshalJSON() {
	vecClock, _ := NewClockBuilder().Id(1).Length(5).Build()
	vecClock.TickLocal() // vecClock.String() == "[1, 0, 0, 0, 0]"

	ts, _ := json.Marshal(vecClock)
	fmt.Println("vecClock JSON:", string(ts))
	// Output: vecClock JSON: {"id":1,"v":["1","0","0","0","0"]}
}

func ExampleTimestamp_ClockBase() {
	ts := Timestamp{
		Id:     1,
		Vector: []string{"9", "10", "11", "12"},
	}

	vecClock, _ := ts.ClockBase(10)
	fmt.Println(vecClock)
	// Output: [9, 10, 11, 12]
}

func ExampleClock_TickReceive() {
	clkA, _ := NewClockBuilder().Id(2).Length(4).Build()
	clkB, _ := NewClockBuilder().Id(3).Length(4).Build()

	fmt.Println("message sent from A to B")
	clkA.TickLocal()              // clkA: [0, 1, 0, 0]
	err := clkB.TickReceive(clkA) // clkB: [0, 1, 0, 0]
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("A:", clkA)
	fmt.Println("B:", clkB)

	fmt.Println("message sent from B to A")
	clkB.TickLocal()             // clkB: [0, 1, 1, 0]
	err = clkA.TickReceive(clkB) // clkA: [0, 1, 1, 0]
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("A:", clkA)
	fmt.Println("B:", clkB)
	// Output:
	// message sent from A to B
	// A: [0, 1, 0, 0]
	// B: [0, 1, 0, 0]
	// message sent from B to A
	// A: [0, 1, 1, 0]
	// B: [0, 1, 1, 0]
}

func TestMessageReceptacle_NewMessageReceptacle(t *testing.T) {
	clk1, _ := NewClockBuilder().Id(1).Length(2).Build() // p1's clock
	clk2, _ := NewClockBuilder().Id(2).Length(2).Build() // p2's clock
	rcp := NewMessageReceptacle(2)                       // p0's receptacle

	// check the initial state of rcp
	if rcp.Length() != 2 {
		t.Fatal("length should be 2")
	}
	if rcp.Size() != 0 {
		t.Fatal("size should be 0")
	}
	if ToString(rcp.counter) != "[0 0]" {
		t.Fatal("rcp.counter.String() should be [0 0]")
	}
	delivery, err, badMsg := rcp.Deliverables()
	if err != nil || badMsg != nil {
		t.Fatal("err and badMsg should be nil")
	}
	if len(delivery) != 0 {
		t.Fatalf("delivery length should be 0, GOT: %d", len(delivery))
	}

	// (1) p1 -> p2
	clk1.TickLocal()
	// clk1 (send): [1, 0]
	if !(clk1.String() == "[1, 0]") {
		t.Fatal(`clk1.String() != "[1, 0]"`)
	}
	msg1 := NewMessage("hey p2! didgeridoo and you can too!", clk1)
	msg1Bytes, _ := json.Marshal(msg1)

	// p2 receives p1's first message
	p2Receipt := new(Message)
	_ = json.Unmarshal(msg1Bytes, p2Receipt)
	p2ReceiptClock, _ := p2Receipt.Timestamp.Clock()
	clk2.TickReceive(p2ReceiptClock)
	// clk2 (receive): [1, 0]
	if !(clk2.String() == "[1, 0]") {
		t.Fatal(`clk2.String() != "[1, 0]"`)
	}

	// (2) p2 -> p1
	clk2.TickLocal()
	// clk2 (send): [1, 1]
	if !(clk2.String() == "[1, 1]") {
		t.Fatal(`clk2.String() != "[1, 1]"`)
	}
	msg2 := NewMessage("hey p1! what's a didgeridoo?!", clk2)
	msg2Bytes, _ := json.Marshal(msg2)

	// p0 receives p2's message
	p0Receipt := new(Message)
	_ = json.Unmarshal(msg2Bytes, p0Receipt)
	_ = rcp.Receive(p0Receipt)
	// check the state of rcp
	if rcp.Length() != 2 {
		t.Fatal("length should be 2")
	}
	if rcp.Size() != 1 {
		t.Fatal("size should be 1")
	}
	if ToString(rcp.counter) != "[0 0]" {
		t.Fatal("rcp.counter.String() should be [0 0]")
	}
	delivery, err, badMsg = rcp.Deliverables()
	if err != nil || badMsg != nil {
		t.Fatal("err and badMsg should be nil")
	}
	if len(delivery) != 0 {
		t.Fatalf("delivery length should be 0, GOT: %d", len(delivery))
	}

	// p0 receives p1's first message
	p0Receipt = new(Message)
	_ = json.Unmarshal(msg1Bytes, p0Receipt)
	_ = rcp.Receive(p0Receipt)
	// check the state of rcp
	if rcp.Length() != 2 {
		t.Fatal("length should be 2")
	}
	if rcp.Size() != 2 {
		t.Fatal("size should be 2")
	}
	delivery, err, badMsg = rcp.Deliverables()
	if err != nil || badMsg != nil {
		t.Fatal("err and badMsg should be nil")
	}
	if len(delivery) != 1 {
		t.Fatalf("delivery length should be 1, GOT: %d", len(delivery))
	}
	delivery, err, badMsg = rcp.Deliverables()
	if err != nil || badMsg != nil {
		t.Fatal("err and badMsg should be nil")
	}
	if len(delivery) != 1 {
		t.Fatalf("delivery length should be 1, GOT: %d", len(delivery))
	}
	if ToString(rcp.counter) != "[1 1]" {
		t.Fatal("rcp.counter.String() should be [1 1]")
	}

	// p1 receives p2's message
	p1Receipt := new(Message)
	_ = json.Unmarshal(msg2Bytes, p1Receipt)
	p1ReceiptClock, _ := p1Receipt.Timestamp.Clock()
	clk1.TickReceive(p1ReceiptClock)
	// clk1 (receive): [1, 1]
	if !(clk1.String() == "[1, 1]") {
		t.Fatal(`clk1.String() != "[1, 1]"`)
	}

	// (3) p1 executes a local event
	clk1.TickLocal()
	// clk1 (report): [2, 1]
	if !(clk1.String() == "[2, 1]") {
		t.Fatal(`clk1.String() != "[2, 1]"`)
	}
	msg3 := NewMessage("hey p0! I did the thing!", clk1)
	msg3Bytes, _ := json.Marshal(msg3)

	// p0 receives p1's second message
	p0Receipt = new(Message)
	_ = json.Unmarshal(msg3Bytes, p0Receipt)
	_ = rcp.Receive(p0Receipt)
	// check the state of rcp
	if rcp.Length() != 2 {
		t.Fatal("length should be 2")
	}
	if rcp.Size() != 1 {
		t.Fatal("size should be 1")
	}
	delivery, err, badMsg = rcp.Deliverables()
	if err != nil || badMsg != nil {
		t.Fatal("err and badMsg should be nil")
	}
	if len(delivery) != 1 {
		t.Fatalf("delivery length should be 1, GOT: %d", len(delivery))
	}
	if ToString(rcp.counter) != "[2 1]" {
		t.Fatal("rcp.counter.String() should be [2 1]")
	}
}

func ToString(counter []logical.Clock) string {
	return "[" + counter[0].String() + " " + counter[1].String() + "]"
}

func ExampleClock_LessThan() {
	clkA, _ := NewClockBuilder().Id(1).Length(3).Build()
	clkB, _ := NewClockBuilder().Id(3).Length(3).Build()
	clkA.TickLocal() // [1, 0, 0]
	clkB.TickLocal() // [0, 0, 1]

	fmt.Println("[1, 0, 0] < [0, 0, 1]:", clkA.LessThan(clkB))
	fmt.Println("[0, 0, 1] < [1, 0, 0]:", clkB.LessThan(clkA))

	clkB.TickReceive(clkA) // clkB: [0, 0, 2]
	fmt.Println("[1, 0, 0] < [0, 0, 2]:", clkA.LessThan(clkB))
	fmt.Println("[0, 0, 2] < [1, 0, 0]:", clkB.LessThan(clkA))
	// Output:
	// [1, 0, 0] < [0, 0, 1]: false
	// [0, 0, 1] < [1, 0, 0]: false
	// [1, 0, 0] < [0, 0, 2]: true
	// [0, 0, 2] < [1, 0, 0]: false
}

func ExampleClock_Concurrent() {
	clkA, _ := NewClockBuilder().Id(1).Length(3).Build()
	clkB, _ := NewClockBuilder().Id(3).Length(3).Build()
	clkA.TickLocal() // [1, 0, 0]
	clkB.TickLocal() // [0, 0, 1]

	fmt.Println(clkA.Concurrent(clkB))
	// Output: true
}

func ExampleMessage() {
	message := Message{
		Content: "didgeridoo",
		Timestamp: Timestamp{
			Id:     1,
			Vector: []string{"0", "0", "1"},
		},
	}
	b, _ := json.Marshal(message)
	fmt.Println(string(b))

	msgData := []byte(`{"msg":"didgeridoo","ts":{"id":1,"v":["0","0","1"]}}`)
	fmt.Println("bytes.Equal(b, msgData):", bytes.Equal(b, msgData))
	// Output:
	// {"msg":"didgeridoo","ts":{"id":1,"v":["0","0","1"]}}
	// bytes.Equal(b, msgData): true
}
