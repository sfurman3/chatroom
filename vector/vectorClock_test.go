package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestString(t *testing.T) {
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

	clk, _ = NewClock().Length(1).Id(1).Build()
	if clk.String() != "[0]" {
		fmt.Println(clk.String(), "!= \"[0]\"")
		t.Fail()
	}

	clk, _ = NewClock().Length(2).Id(1).Build()
	if clk.String() != "[0, 0]" {
		fmt.Println(clk.String(), "!= \"[0, 0]\"")
		t.Fail()
	}
}

func ExampleUnmarshalJSON() {
	vecClockJsonBytes := []byte(`{"id":1,"v":["1","0","0","0","0"]}`)

	vecClock := new(Clock)
	_ = json.Unmarshal(vecClockJsonBytes, vecClock)
	fmt.Println("vecClock:", vecClock)
	// Output: vecClock: [1, 0, 0, 0, 0]
}

func ExampleMarshalJSON() {
	vecClock, _ := NewClock().Id(1).Length(5).Build()
	vecClock.TickLocal() // vecClock.String() == "[1, 0, 0, 0, 0]"

	ts, _ := json.Marshal(vecClock)
	fmt.Println("vecClock JSON:", string(ts))
	// Output: vecClock JSON: {"id":1,"v":["1","0","0","0","0"]}
}

func ExampleVectorClock() {
	ts := Timestamp{
		Id:     1,
		Vector: []string{"9", "10", "11", "12"},
	}

	vecClock, _ := ts.VectorClock(10)
	fmt.Println(vecClock)
	// Output: [9, 10, 11, 12]
}

func ExampleLessThan() {
	clkA, _ := NewClock().Id(1).Length(3).Build()
	clkB, _ := NewClock().Id(3).Length(3).Build()
	clkA.TickLocal() // [1, 0, 0]
	clkB.TickLocal() // [0, 0, 1]

	fmt.Println("[1, 0, 0] < [0, 0, 1]:", clkA.LessThan(clkB))
	fmt.Println("[0, 0, 1] < [1, 0, 0]:", clkB.LessThan(clkA))

	clkB.TickReceive(clkA) // clkB: [1, 0, 2]
	fmt.Println("[1, 0, 0] < [1, 0, 2]:", clkA.LessThan(clkB))
	fmt.Println("[1, 0, 2] < [1, 0, 0]:", clkB.LessThan(clkA))
	// Output:
	// [1, 0, 0] < [0, 0, 1]: false
	// [0, 0, 1] < [1, 0, 0]: false
	// [1, 0, 0] < [1, 0, 2]: true
	// [1, 0, 2] < [1, 0, 0]: false
}

func ExampleConcurrent() {
	clkA, _ := NewClock().Id(1).Length(3).Build()
	clkB, _ := NewClock().Id(3).Length(3).Build()
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
