package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
)

type temperature struct {
	min, max, sum float64
	num           int
}

// Small enough to avoid exhausting memory, but
// large enough to minimize overhead from creating and managing goroutines.
// 1 million rows
const batchSize = 1000 * 1000
const chunkSize = 4096 * 1000 // 4 MB chunks
// Too large => more memory
// Too small => slow perf
const chanSize = 1000         // 1000 * 1 million = 1 billion

var dataChan = make(chan [][]byte, chanSize)
var resultMap sync.Map

func processLine(line []byte) {
	if len(line) == 0 {
		return
	}
	splitIndex := bytes.IndexByte(line, ';')
	if splitIndex == -1 {
		return
	}
	station := string(line[:splitIndex])
	reading, err := strconv.ParseFloat(string(line[splitIndex+1:]), 64)
	if err != nil {
		return
	}
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
	// CPU profiling
	cpuFile, err := os.Create("cpu_profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// Memory profiling
	memFile, err := os.Create("mem_profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer memFile.Close()

	tempErr := setTemperatureReading("measurements.txt")
	if tempErr != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	writeToFile(formatOutput())

	// Write memory profile
	runtime.GC() // Run a garbage collection to get up-to-date statistics
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Profiling complete. CPU profile saved to cpu_profile.prof and memory profile saved to mem_profile.prof")
}
