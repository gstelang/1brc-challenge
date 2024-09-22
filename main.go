package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
)

type temperature struct {
	min, max, sum float64
	num           int
}

type temperatureReading struct {
	station []byte
	value   float64
}

// Small enough to avoid exhausting memory, but
// large enough to minimize overhead from creating and managing goroutines.
// 1 million rows
const batchSize = 1000 * 1000
const chunkSize = 4096 * 1000 // 4 MB chunks
const chanSize = 1000         // 1000 * 1 million = 1 billion

var dataChan = make(chan [][]byte, chanSize)
var readingsChan = make(chan temperatureReading, batchSize)

var resultMap = make(map[string]temperature)

func aggregateTemperatures(aggWg *sync.WaitGroup) {
	defer aggWg.Done()

	for reading := range readingsChan {
		currentVal := reading.value
		currentStation := string(reading.station)
		temp, exists := resultMap[currentStation]
		if !exists {
			resultMap[currentStation] = temperature{
				min: currentVal,
				max: currentVal,
				sum: currentVal,
				num: 1,
			}
		} else {
			resultMap[currentStation] = temperature{
				min: min(temp.min, currentVal),
				max: max(temp.max, currentVal),
				sum: temp.sum + currentVal,
				num: temp.num + 1,
			}
		}
	}

}

func processLine(line []byte) {
	if len(line) == 0 {
		return
	}
	splitIndex := bytes.IndexByte(line, ';')
	if splitIndex == -1 {
		return
	}
	station := line[:splitIndex]
	reading, err := strconv.ParseFloat(string(line[splitIndex+1:]), 64)
	if err != nil {
		return
	}
	readingsChan <- temperatureReading{
		station: station,
		value:   reading,
	}
}

func processBatch(wg *sync.WaitGroup) {
	defer wg.Done()
	for _, line := range <-dataChan {
		processLine(line)
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

	batch := make([][]byte, 0, batchSize)
	buffer := make([]byte, chunkSize)
	lastLineRead := make([]byte, 0)

	var aggWg sync.WaitGroup
	aggWg.Add(1)
	go aggregateTemperatures(&aggWg)

	var wg sync.WaitGroup
	wg.Add(1)
	go processBatch(&wg)

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading file:", err)
			}
			break
		}

		// bytes read will be either size of chunk defined or less or less.
		data := buffer[:bytesRead]
		ans := bytes.Split(append(lastLineRead, data...), []byte("\n"))
		batch = append(batch, ans[:len(ans)-1]...)

		if len(batch) >= batchSize {
			// Send batch to the channel
			select {
			case dataChan <- batch:
				// Successfully sent the batch, clear the batch
				batch = batch[:0]
			default:
				wg.Add(1)
				go processBatch(&wg)
				dataChan <- batch
				batch = batch[:0]
			}
		}

		lastLineRead = ans[len(ans)-1]
	}

	if len(lastLineRead) != 0 {
		batch = append(batch, lastLineRead)
	}

	// Process any remaining lines
	if len(batch) > 0 {
		wg.Add(1)
		go processBatch(&wg)
		dataChan <- batch
	}

	close(dataChan)
	wg.Wait()
	close(readingsChan)
	aggWg.Wait()
	return nil
}

func formatOutput() string {
	out := "{"
	total := len(resultMap)
	count := 0
	for key, value := range resultMap {
		// min, mean, max in this order
		formattedMin := fmt.Sprintf("%.1f", value.min)
		formattedMean := fmt.Sprintf("%.1f", value.sum/float64(value.num))
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
	err := setTemperatureReading("measurements.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	output := formatOutput()
	writeToFile(output)
}
