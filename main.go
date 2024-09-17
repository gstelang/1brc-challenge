package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type temperature struct {
	min float64
	max float64
	sum float64
	num float64
}

var resultMap = make(map[string]temperature)
var mu sync.Mutex
var dataChan = make(chan string, 1000000)

// Worker function to read from the channel and add it to the map
func worker(dataChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for line := range dataChan {
		mu.Lock()
		splitStr := strings.Split(line, ";")
		station := splitStr[0]
		reading, _ := strconv.ParseFloat(splitStr[1], 64)

		val, ok := resultMap[station]

		if ok {
			resultMap[station] = temperature{
				max: max(val.max, reading),
				min: min(val.min, reading),
				num: val.num + 1,
				sum: reading + val.sum,
			}

		} else {
			resultMap[station] = temperature{
				max: reading,
				min: reading,
				num: 1,
				sum: reading,
			}

		}
		mu.Unlock()
	}
}

func writeToFile(str string) {
	file, err := os.Create("output.txt")
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(str)
	if err != nil {
		fmt.Println("Failed to write to file:", err)
		return
	}

	fmt.Println("Wrote to file 'output.txt'.")
}

func setTemperatureReading(fileLocation string) error {

	file, err := os.Open(fileLocation)

	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()

	// WaitGroup for processing lines
	var wg sync.WaitGroup

	scanner := bufio.NewScanner(file)

	// Start workers to consume data from channel and add to the map
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(dataChan, &wg)
	}

	// Read the file line by line
	for scanner.Scan() {
		line := scanner.Text()
		dataChan <- line
	}

	close(dataChan)

	// Wait for all workers to finish processing the data from the channel
	wg.Wait()

	return nil
}

func formatOutput(measurementMap map[string]temperature) string {
	out := "{"
	total := len(measurementMap)
	count := 0
	for key, value := range measurementMap {
		// min, mean, max in this order
		formattedMin := fmt.Sprintf("%.1f", value.min)
		formattedMean := fmt.Sprintf("%.1f", value.sum/value.num)
		formattedMax := fmt.Sprintf("%.1f", value.max)

		out += key + "=" + formattedMin + "/" + formattedMean + "/" + formattedMax
		count++

		if count != total {
			out += ", "
		} else {
			out += "}"
		}
	}
	return out
}

func main() {
	setTemperatureReading("measurements.txt")
	output := formatOutput(resultMap)
	writeToFile(output)
}
