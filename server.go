package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var (
	ID                 = -1 // id of the process {0, ..., NUM_PROCS-1}
	NUM_PROCS          = -1 // total number of processes
	PORT               = -1 // number of the master-facing port
	REQUIRED_ARGUMENTS = []*int{&ID, &NUM_PROCS, &PORT}
)

func init() {
	flag.IntVar(&ID, "id", ID, "id of the process {0, ..., n-1}")
	flag.IntVar(&NUM_PROCS, "n", NUM_PROCS, "total number of processes")
	flag.IntVar(&PORT, "port", PORT, "number of the master-facing port")
	flag.Parse()

	setArgsPositional()
}

func main() {
	fmt.Println(ID)
	fmt.Println(NUM_PROCS)
	fmt.Println(PORT)
}

func setArgsPositional() {
	getIntArg := func(i int) int {
		arg := flag.Arg(i)
		if arg == "" {
			fmt.Fprintln(os.Stderr, os.Args)
			fmt.Fprintln(os.Stderr, "missing one or more arguments\n")
			flag.PrintDefaults()
			os.Exit(1)
		}
		val, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Printf("could not parse: '%v' into an integer\n", arg)
			os.Exit(1)
		}
		return val
	}

	for idx, val := range REQUIRED_ARGUMENTS {
		if *val == -1 {
			*val = getIntArg(idx)
		}
	}
}
