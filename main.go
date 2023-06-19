package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/montanaflynn/stats"
)

type TTAA struct {
	TimeStamp time.Time
	CId       int64
	StartTs   int64
	Duration  int64
}

type CustomerSummary struct {
	CId    int64
	Count  int
	Mean   float64
	Median float64
	P99    float64
}

type CustomerReport struct {
	TimeStamp time.Time
	Customers map[int64]CustomerSummary
	Digests   map[int64]*tdigest.TDigest
	Count     int
	Mean      float64
	Median    float64
	P99       float64
}

var (
	debug = flag.Int64("debug", 0, "Customer ID to debug")
)

func loadData(f string) []TTAA {
	var err error
	ttaa := []TTAA{}
	//var newCustId int64 = 1
	//idMap := make(map[int64]int64)

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
		if t.Duration > 11093887505 {
			//bad data
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
		// Remap IDs so they don't match any "real" data ever
		//if newId, ok := idMap[t.CId]; !ok {
		//	idMap[t.CId] = newCustId
		//	t.CId = newCustId
		//	newCustId++
		//} else {
		//	t.CId = newId
		//}

		t.TimeStamp, err = time.Parse(time.RFC3339Nano, strings.Trim(columns[0], "\""))
		if err != nil {
			log.Printf("Parse error for @timestamp column: %s", row)
			continue
		}
		ttaa = append(ttaa, t)
	}

	return ttaa
}

// filterDurations returns a slice of int64 Durations for TTAA records that
// match the given customerId or all records if CId is -1.
func filterDurations(r []TTAA, CId int64) []float64 {
	array := []float64{}
	for _, t := range r {
		if t.CId == CId || CId == -1 {
			array = append(array, float64(t.Duration))
		}
	}

	return array
}

// buildSummary extracs TTAA records for the given Customer ID (or -1 for all)
// and creates a CustomerSummary rollup object with the matching records.
func buildSummary(data []TTAA, CId int64) (c CustomerSummary) {
	var err error

	c.CId = CId
	durations := filterDurations(data, CId)
	c.Count = len(durations)
	c.Mean, err = stats.Mean(durations)
	if err != nil {
		log.Fatal(err)
	}
	c.Median, err = stats.Median(durations)
	if err != nil {
		log.Fatal(err)
	}
	c.P99, err = stats.Percentile(durations, 99)
	if err != nil {
		log.Fatal(err)
	}
	if *debug > 0 && c.CId == *debug {
		log.Printf("%d: count: %d, mu: %.2f, p99: %.2f", *debug, c.Count, c.Mean, c.P99)
	}

	return
}

func buildDigest(data []TTAA, CId int64) *tdigest.TDigest {
	t := tdigest.New()
	for _, v := range filterDurations(data, CId) {
		t.Add(float64(v), 1.0)
	}

	return t
}

// reportCustomerSummaries takes a map of unique Customer IDs each with one
// CustomerSummary and logs it to screen and writes it to a CSV file.
func reportCustomerSummaries(cust CustomerReport, cDigests map[int64]*tdigest.TDigest, filename string) {
	fd, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	if len(cust.Customers) != len(cDigests)-1 {
		log.Fatalf("Number of unique customers mismatch: %d:%d", len(cust.Customers), len(cDigests)-1)
	}

	fd.WriteString("CId,Mean,Median,P99,TMedian,TP99,Error,Count\n")
	e := 100.0 * (math.Abs(cust.P99 - cDigests[-1].Quantile(.99))) / cust.P99
	fmt.Fprintf(fd, "%d,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
		-1, cust.Mean, cust.Median, cust.P99,
		cDigests[-1].Quantile(0.5), cDigests[-1].Quantile(0.99),
		e, cDigests[-1].Count())

	for _, c := range cust.Customers {
		//log.Printf("Customer %d has mu == %.2f, p50 == %.2f, p99 == %.2f", c.CId, c.Mean, c.Median, c.P99)
		e := 100.0 * (math.Abs(cDigests[c.CId].Quantile(.99) - c.P99)) / c.P99
		fmt.Fprintf(fd, "%d,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			c.CId, c.Mean, c.Median, c.P99,
			cDigests[c.CId].Quantile(.5), cDigests[c.CId].Quantile(.99),
			e, cDigests[c.CId].Count())
		if *debug == c.CId {
			log.Printf("%d: %v", c.CId, cDigests[*debug])
			log.Printf("%d: T-Digest Rollup p99 %.2f, Error %.2f", c.CId,
				cDigests[c.CId].Quantile(.99), e)
		}
	}
}

func getCustomers(data []TTAA) (c CustomerReport) {
	c.Customers = make(map[int64]CustomerSummary)
	c.Digests = make(map[int64]*tdigest.TDigest)
	for _, tta := range data {
		if _, ok := c.Customers[tta.CId]; !ok {
			c.Customers[tta.CId] = buildSummary(data, tta.CId)
			c.Digests[tta.CId] = buildDigest(data, tta.CId)
		}
		c.Count++ // Keep count of total number of observations
	}

	return c
}

// partition returns a Unix Epoch time stamp adjusted to be an increment of
// the given seconds parameter.
func partition(t time.Time, seconds int64) int64 {
	return t.Unix() - (t.Unix() % seconds)
}

func rollupData(data []TTAA, seconds int64) []CustomerReport {
	var timeSeries []CustomerReport
	var left = 0
	var right = 0
	var ts = partition(data[0].TimeStamp, seconds)

	rollup := func() {
		customers := getCustomers(data[left:right])
		customers.TimeStamp = time.Unix(ts, 0)
		customers.Mean, _ = stats.Mean(filterDurations(data[left:right], -1))
		customers.Median, _ = stats.Median(filterDurations(data[left:right], -1))
		customers.P99, _ = stats.Percentile(filterDurations(data[left:right], -1), 99)
		//log.Printf("Summary for %v: Customers == %d, mu == %.2f, Median == %.2f, P99 == %.2f",
		//	customers.TimeStamp, len(customers.Customers), customers.Mean, customers.Median, customers.P99)
		timeSeries = append(timeSeries, customers)
	}

	for i, tta := range data {
		if ts != partition(tta.TimeStamp, seconds) {
			right = i
			rollup()
			left = i
			ts = partition(tta.TimeStamp, seconds)
		}
	}
	right = len(data)
	if right > left {
		rollup()
	}

	return timeSeries
}

func buildTDigests(ts []CustomerReport) map[int64]*tdigest.TDigest {
	var cDigests = make(map[int64]*tdigest.TDigest)
	var count int

	cDigests[-1] = tdigest.New()
	for _, batch := range ts {
		for CId, c := range batch.Digests {
			if _, ok := cDigests[CId]; !ok {
				cDigests[CId] = tdigest.New()
			}
			count++

			// Create the 24h TDigest from all the rollup TDigests
			cDigests[CId].Merge(c)
			// Build a TDigest for all customers with cID == -1
			cDigests[-1].Merge(c)
			if *debug == CId {
				log.Printf("%d: %v", CId, c)
			}
		}
		//for _, c := range batch.Customers {
		//	if _, ok := cDigests[c.CId]; !ok {
		//		cDigests[c.CId] = tdigest.New()
		//	}
		//	count++
		//	cDigests[c.CId].Add(c.Mean, float64(c.Count))

		//	// Build a TDigest for all customers with cID == -1
		//	cDigests[-1].Add(c.Mean, float64(c.Count))
		//}
	}

	// Now we have TDigests for each customer and all customers that
	// represents the entire set of test data reconstructed from the 5m
	// rollups of that data.
	log.Printf("T-Digest Rollup: %d merges", count)
	log.Printf("T-Digest Rollup: Found %d unique customers", len(cDigests)-1)
	log.Printf("T-Digest Rollup: All observations   : %v", cDigests[-1].Count())
	//log.Printf("T-Digest: All customer Mean  : %.2f", cDigests[-1].TrimmedMean(0, 1))
	log.Printf("T-Digest Rollup: All customer Median: %.2f", cDigests[-1].Quantile(0.5))
	log.Printf("T-Digest Rollup: All customer p99   : %.2f", cDigests[-1].Quantile(0.99))
	return cDigests
}

func main() {
	// Setup and parse the command line flags
	csv := flag.String("csv", "/dev/null", "CSV file to parse")
	interval := flag.Int64("interval", 120, "Rollup interval to use in seconds")
	flag.Parse()

	rawData := loadData(*csv)
	if len(rawData) == 0 {
		log.Fatalf("Found no TTAA data in %s", *csv)
	}
	timeSeries10d := rollupData(rawData, 864000) // 10 Days -- make sure all the data is in one report, we've got about 24h of raw data
	if len(timeSeries10d) != 1 {
		log.Fatalf("All the raw data should fit into the first report!")
	}

	timeSeries5m := rollupData(rawData, *interval) // 5m rollups for each customer found in the [5m] window
	cDigests := buildTDigests(timeSeries5m)
	reportCustomerSummaries(timeSeries10d[0], cDigests, "tdigest-results.csv")

	// Compare to a T-Digest over the raw data
	rawTDigest := tdigest.New()
	for _, tta := range rawData {
		rawTDigest.Add(float64(tta.Duration), 1)
	}
	log.Printf("T-Digest Raw Data: All customer Median: %.2f", rawTDigest.Quantile(0.5))
	log.Printf("T-Digest Raw Data: All customer p99   : %.2f", rawTDigest.Quantile(0.99))

	log.Printf("Found %d TTAA records", len(rawData))
	log.Printf("Found %d unique customers", len(timeSeries10d[0].Customers))
	log.Printf("All customer Mean  : %.2f", timeSeries10d[0].Mean)
	log.Printf("All customer Median: %.2f", timeSeries10d[0].Median)
	log.Printf("All customer p99   : %.2f", timeSeries10d[0].P99)
	log.Printf("Bytes Required for each T-Digest serialization: %d", 4000)
}
