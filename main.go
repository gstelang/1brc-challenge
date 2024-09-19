package main

import (
	"fmt"
	"io"
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

// Small enough to avoid exhausting memory, but
// large enough to minimize overhead from creating and managing goroutines.
const batchSize = 1000000

var resultMap sync.Map

// Worker function to process a batch of data
func processBatch(batch []string, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, line := range batch {
		delimiterIndex := strings.Index(line, ";")
		if delimiterIndex == -1 {
			// skip line
			continue
		}
		station := line[:delimiterIndex]
		reading, _ := strconv.ParseFloat(line[delimiterIndex+1:], 64)

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

	var wg sync.WaitGroup
	batch := make([]string, 0, batchSize)
	chunkSize := 4096 * 1000 // 4 MB chunks
	buffer := make([]byte, chunkSize)
	lastLineRead := ""

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading file:", err)
			}
			break
		}

		// bytes read will be either size of chunk defined or less or less.
		str := string(buffer[:bytesRead])
		ans := strings.Split(lastLineRead+str, "\n")
		batch = append(batch, ans[:len(ans)-1]...)

		if len(batch) >= batchSize {
			wg.Add(1)
			// Make a copy of the batch.
			// if we pass batch, it can result into data race where we're processing for next batch while goroutine is modifying the previous batch.
			go processBatch(append([]string(nil), batch...), &wg)
			batch = batch[:0] // Clear the batch
		}

		lastLineRead = ans[len(ans)-1]
	}

	if lastLineRead != "" {
		batch = append(batch, lastLineRead)
	}

	// Process any remaining lines
	if len(batch) > 0 {
		wg.Add(1)
		go processBatch(append([]string(nil), batch...), &wg)
	}

	// Wait for all batches to be processed
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
	err := setTemperatureReading("measurements.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	output := formatOutput()
	writeToFile(output)
}
