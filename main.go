package main

/*
#include <sys/mman.h>
#include <fcntl.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>

void* mmap_file(const char* filename, size_t length, off_t offset, size_t* filesize) {
    int fd = open(filename, O_RDONLY);
    if (fd == -1) {
        perror("open");
        return NULL;
    }

    // Get the file size
    *filesize = lseek(fd, 0, SEEK_END);
    if (*filesize == (size_t)-1) {
        perror("lseek");
        close(fd);
        return NULL;
    }

    // Reset file pointer to beginning
    if (lseek(fd, 0, SEEK_SET) == (off_t)-1) {
        perror("lseek");
        close(fd);
        return NULL;
    }

    // Adjust length if it's 0 (map whole file) or exceeds file size
    if (length == 0 || offset + length > *filesize) {
        length = *filesize - offset;
    }

    // Map the file
    void* map = mmap(NULL, length, PROT_READ, MAP_PRIVATE, fd, offset);
    close(fd);

    if (map == MAP_FAILED) {
        perror("mmap");
        return NULL;
    }

    // Advise the kernel of our access pattern
    if (madvise(map, length, MADV_SEQUENTIAL | MADV_WILLNEED) != 0) {
        perror("madvise");
    }

    return map;
}

int munmap_file(void* map, size_t length) {
    if (madvise(map, length, MADV_DONTNEED) != 0) {
        perror("madvise");
    }
    return munmap(map, length);
}
*/
import "C"
import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"sync"
	"unsafe"
)

func doCalculations(fileLocation string) {
	// File to map
	filename := C.CString(fileLocation)
	defer C.free(unsafe.Pointer(filename))

	var filesize C.size_t

	// Get file size from mmap_file
	C.mmap_file(filename, 0, 0, &filesize)

	// Get the system page size (this is necessary for proper alignment)
	pageSize := C.size_t(C.sysconf(C._SC_PAGESIZE))
	fmt.Printf("Page size: %d bytes\n", pageSize)

	// Define the chunk size (e.g., 4KB or larger than page size)
	chunkSize := pageSize * 1000 // Ensure the chunk size is multiple of the page size

	batch := make([][]byte, 0, batchSize)
	lastLineRead := make([]byte, 0)
	var aggWg sync.WaitGroup
	aggWg.Add(1)
	go aggregateTemperatures(&aggWg)

	var wg sync.WaitGroup
	wg.Add(1)
	go processBatch(&wg)

	// Loop through the file in chunks
	for offset := C.off_t(0); offset < C.off_t(filesize); offset += C.off_t(chunkSize) {
		// Adjust the length for the last chunk
		length := chunkSize
		if C.off_t(filesize)-offset < C.off_t(chunkSize) {
			length = C.size_t(C.off_t(filesize) - offset)
		}

		// Find the nearest previous page boundary for the offset
		alignedOffset := offset - (offset % C.off_t(pageSize))
		offsetDifference := offset - alignedOffset

		// Map the file chunk from the aligned offset
		mappedData := C.mmap_file(filename, length+C.size_t(offsetDifference), alignedOffset, &filesize)
		if mappedData == nil {
			fmt.Println("Error mapping file at offset", offset)
			return
		}

		// Convert the mapped portion to a Go byte slice, starting from the correct offset
		data := C.GoBytes(unsafe.Pointer(uintptr(unsafe.Pointer(mappedData))+uintptr(offsetDifference)), C.int(length))
		ans := bytes.Split(append(lastLineRead, data...), []byte("\n"))
		batch = append(batch, ans[:len(ans)-1]...)
		lastLineRead = ans[len(ans)-1]

		if len(batch) >= batchSize {
			// Send batch to the channel
			select {
			case dataChan <- batch:
				batch = make([][]byte, 0, batchSize)
			default:
				wg.Add(1)
				go processBatch(&wg)
				dataChan <- batch
				batch = make([][]byte, 0, batchSize)
			}
		}

		// Unmap the chunk
		C.munmap_file(mappedData, length+C.size_t(offsetDifference))
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

	wg.Wait()
}

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
	doCalculations("measurements.txt")
	output := formatOutput()
	writeToFile(output)
}
