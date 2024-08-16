package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type temperature struct {
	min float64
	max float64
	sum float64
	num float64
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

func getTemperatureReading(fileLocation string) map[string]temperature {
	measurementMap := make(map[string]temperature)

	file, err := os.Open(fileLocation)
	if err != nil {
		fmt.Println(err)
		return measurementMap
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read the file line by line
	for scanner.Scan() {
		line := scanner.Text()

		splitStr := strings.Split(line, ";")
		station := splitStr[0]
		reading, _ := strconv.ParseFloat(splitStr[1], 64)

		val, ok := measurementMap[station]

		if ok {
			measurementMap[station] = temperature{
				max: max(val.max, reading),
				min: min(val.min, reading),
				num: val.num + 1,
				sum: reading + val.sum,
			}

		} else {
			measurementMap[station] = temperature{
				max: reading,
				min: reading,
				num: 1,
				sum: reading,
			}

		}
	}

	return measurementMap
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
	measurementMap := getTemperatureReading("measurements-short.txt")
	output := formatOutput(measurementMap)
	writeToFile(output)
}
