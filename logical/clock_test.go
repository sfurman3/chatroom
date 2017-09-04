package logical

import (
	"fmt"
	"testing"
)

func TestTickZero(t *testing.T) {
	var clk Clock
	clk.Tick()

	if clk.Text(10) != "1" {
		t.Fail()
	}
}

func TestSetStringZero(t *testing.T) {
	clk, succ := new(Clock).SetString("0", 10)
	if !succ {
		t.Fail()
	}

	if !(clk.Text(10) == "0") {
		t.Fail()
	}
}

func TestSetStringNegativeFails(t *testing.T) {
	_, succ := new(Clock).SetString("-1", 10)
	if succ {
		t.Fail()
	}
}

func TestTickZeroValue(t *testing.T) {
	clk, _ := new(Clock).SetString("0", 10)
	clk.Tick()

	if clk.Text(10) != "1" {
		t.Fail()
	}
}

func TestTickReceiveZero(t *testing.T) {
	var clk Clock
	clk.TickReceive(new(Clock))
	if clk.Text(10) != "1" {
		fmt.Println(clk.Text(10))
		t.Fail()
	}
}

func TestTickReceiveOne(t *testing.T) {
	clk := new(Clock)
	other, _ := new(Clock).SetString("1", 10)
	clk.TickReceive(other)
	if clk.Text(10) != "2" {
		fmt.Println(clk.Text(10))
		t.Fail()
	}
}

func TestTickReceiveOneOtherNil(t *testing.T) {
	clk, _ := new(Clock).SetString("1", 10)
	clk.TickReceive(new(Clock))
	if clk.Text(10) != "2" {
		fmt.Println(clk.Text(10))
		t.Fail()
	}
}

func TestCmpZeroClock(t *testing.T) {
	clk := new(Clock)
	other := new(Clock)
	if !(clk.Cmp(other) == 0) {
		t.Fail()
	}
}

func TestCmpClockToZero(t *testing.T) {
	clk := new(Clock)
	other, _ := new(Clock).SetString("0", 10)
	if !(clk.Cmp(other) == 0) {
		t.Fail()
	}
}

func TestCmpZeroToClock(t *testing.T) {
	clk := new(Clock)
	other, _ := new(Clock).SetString("0", 10)
	if !(other.Cmp(clk) == 0) {
		t.Fail()
	}
}
