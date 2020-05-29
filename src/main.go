package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"./player"
)

func runLocally(programPath, inputPath, configPath string) {
	numberOfParties := 3
	threshold := 1
	prime := int64(4001)
	if configPath != "" {
		file, err := os.Open(configPath)
		if err != nil && os.IsExist(err) { //Continue execution if file does not exist
			log.Fatal(err)
		}
		if err == nil {
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				tokens := strings.Split(line, " ")
				if len(tokens) != 2 {
					continue
				}
				switch tokens[0] {
				case "p":
					value, err := strconv.ParseInt(tokens[1], 10, 64)
					if err == nil {
						prime = value
					}
				case "n":
					value, err := strconv.Atoi(tokens[1])
					if err == nil {
						numberOfParties = value
					}
				case "t":
					value, err := strconv.Atoi(tokens[1])
					if err == nil {
						threshold = value
					}
				}
			}

			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}

	}

	parties := player.LocalSetup(prime, threshold, numberOfParties, programPath, inputPath)
	for i, party := range parties {
		if i == 1 {
			continue
		}
		go party.Run()
	}
	output := parties[1].Run()
	for id, val := range output {
		fmt.Println(id, val)
	}
}

func main() {
	var directory string = "player/tests/compiled/"
	programPath := directory + "prog"
	inputPath := directory + "input"
	configPath := directory + "config"
	if len(os.Args) == 4 {
		programPath = os.Args[1]
		inputPath = os.Args[2]
		configPath = os.Args[3]
	}
	runLocally(programPath, inputPath, configPath)

}
