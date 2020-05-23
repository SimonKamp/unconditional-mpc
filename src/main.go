package main

import (
	"fmt"

	"./player"
)

func runLocally() {
	var directory string = "player/tests/compiled/"
	numberOfParties := 3
	threshold := 1
	prime := int64(4001)
	programPath := directory + "prog"
	inputPath := directory + "input"
	parties := player.LocalSetup(prime, threshold, numberOfParties, programPath, inputPath)
	for i, party := range parties {
		if i == numberOfParties {
			continue
		}
		go party.Run()
	}
	output := parties[numberOfParties].Run()
	for id, val := range output {
		fmt.Println(id, val)
	}
}

func main() {
	runLocally()
}
