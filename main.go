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

var resultMap sync.Map

var dataChan = make(chan string, 10000)

// Worker function to read from the channel and add it to the map
func worker(dataChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for line := range dataChan {
		splitStr := strings.Split(line, ";")
		station := splitStr[0]
		reading, _ := strconv.ParseFloat(splitStr[1], 64)

		// Load the current value from the map
		if val, ok := resultMap.Load(station); ok {
			// If the station exists, update its temperature record
			temp := val.(temperature)
			updatedTemp := temperature{
				max: max(temp.max, reading),
				min: min(temp.min, reading),
				num: temp.num + 1,
				sum: temp.sum + reading,
			}
			resultMap.Store(station, updatedTemp)
		} else {
			// If the station does not exist, create a new temperature record
			newTemp := temperature{
				max: reading,
				min: reading,
				num: 1,
				sum: reading,
			}
			resultMap.Store(station, newTemp)
		}
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

func formatOutput() string {
	out := "{"

	// sync.Map does not have len... so use Range
	resultMap.Range(func(key, value interface{}) bool {
		k := key.(string)
		v := value.(temperature)

		// Format min, mean, and max values
		formattedMin := fmt.Sprintf("%.1f", float64(v.min))
		formattedMean := fmt.Sprintf("%.1f", float64(v.sum)/float64(v.num))
		formattedMax := fmt.Sprintf("%.1f", float64(v.max))
		out += k + "=" + formattedMin + "/" + formattedMean + "/" + formattedMax + ", "
		return true
	})

	return out[:len(out)-2] + "}"
}

func main() {
	setTemperatureReading("measurements.txt")
	output := formatOutput()
	writeToFile(output)
}
