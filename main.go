package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type TTAA struct {
	TimeStamp time.Time
	CId       int64
	StartTs   int64
	Duration  int64
}

func loadData(f string) []TTAA {
	var err error
	ttaa := []TTAA{}

	file, err := os.Open(f)
	if err != nil {
		log.Fatalf("ERROR: Opening file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		row := scanner.Text()
		if strings.HasPrefix(row, "\"@timestamp") {
			// Header row
			continue
		}
		columns := strings.Split(row, ",")
		if len(columns) != 4 {
			log.Printf("Invalid row: %s", row)
			continue
		}

		t := TTAA{}
		t.Duration, err = strconv.ParseInt(columns[3], 10, 64)
		if err != nil {
			log.Printf("Parse error for duration column: %s", row)
			continue
		}
		t.StartTs, err = strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			log.Printf("Parse error for startTs column: %s", row)
			continue
		}
		t.CId, err = strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			log.Printf("Parse error for cId column: %s", row)
			continue
		}
		t.TimeStamp, err = time.Parse(time.RFC3339Nano, strings.Trim(columns[0], "\""))
		if err != nil {
			log.Printf("Parse error for @timestamp column: %s", row)
			continue
		}
		ttaa = append(ttaa, t)
	}

	return ttaa
}

func main() {
	// Setup and parse the command line flags
	csv := flag.String("csv", "/dev/null", "CSV file to parse")
	flag.Parse()

	rawData := loadData(*csv)
	log.Printf("Found %d TTAA records", len(rawData))
}
