package stats

import (
	"bufio"
	"encoding/json"
	"fmt"
	tm "github.com/buger/gor/stats/terminal"
	"log"
	"os"
	"text/tabwriter"
	"time"
)

// Enable debug logging only if "--verbose" flag passed
func Debug(v ...interface{}) {
	if Settings.Verbose {
		log.Println(v...)
	}
}

// Because its sub-program, Run acts as `main`
func Run() {
	file, err := os.Open(Settings.StatFile)

	if err != nil {
		log.Fatal(err)
	}

	var allStats []*PeriodStats

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		stat := &PeriodStats{}

		json.Unmarshal(scanner.Bytes(), stat)

		allStats = append(allStats, stat)
	}

	for {
		tm.Clear()

		fmt.Println("Gor stats tool\t")
		fmt.Println("Current Time:", time.Now().Format(time.RFC1123))
		fmt.Println("\n\n")

		_, err = file.Seek(-30000, 2)

		scanner := bufio.NewScanner(file)

		if err != nil {
			Debug("Error", err)
			file.Seek(0, 0)
		} else {
			scanner.Scan() // Skip first line since it can be incomplete
		}

		for scanner.Scan() {
			stat := &PeriodStats{}

			json.Unmarshal(scanner.Bytes(), stat)

			found := false

			for i := len(allStats) - 1; i > 0; i-- {
				if stat.Timestamp == allStats[i].Timestamp {
					allStats[i] = stat
					found = true
					break
				}
			}

			if found == false {
				allStats = append(allStats, stat)
			}
		}

		started := 0
		finished := 0

		lastActivity := new(tabwriter.Writer)
		lastActivity.Init(os.Stdout, 0, 20, 10, ' ', 0)
		fmt.Fprintln(lastActivity, "Time\tStarted\tInProgress\tFinished\tAvg Lat\tMax Lat\tMin Lat\t.")

		for i := len(allStats) - 1; i >= 0; i-- {
			stat := allStats[i].TotalStats

			if stat == nil {
				continue
			}

			// Limit to 10 latest records
			if (len(allStats) - i) > 10 {
				break
			}

			started += stat.Count
			finished += stat.Finished

			humanTime := TimeInWords(time.Now().Unix() - allStats[i].Timestamp)

			fmt.Fprintf(lastActivity, "%s\t%d\t%d\t%d\t%f\t%f\t%f\n", humanTime, stat.Count, stat.Count-stat.Finished, stat.Finished, stat.AvgLat, stat.MaxLat, stat.MinLat)
		}

		fmt.Println(tm.Bold("Total stats"))

		totals := new(tabwriter.Writer)
		totals.Init(os.Stdout, 0, 20, 10, ' ', 0)
		fmt.Fprintln(totals, "Time\tStarted\tInProgress\tFinished\t.")
		fmt.Fprintf(totals, "All\t%d\t%d\t%d\n", started, started-finished, finished)
		totals.Flush()

		fmt.Println("\n\n")
		fmt.Println(tm.Bold("Latest activity"))
		lastActivity.Flush()

		err = file.Close()

		if err != nil {
			Debug(err)
		}

		file, err = os.Open(Settings.StatFile)

		if err != nil {
			Debug(err)
		}

		time.Sleep(time.Second)
	}

}
