package vector

import (
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
