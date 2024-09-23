# Goal 
* Process 1 billion rows in a file of size 12.85 GB in Go with high perf! Original Repo [here](https://github.com/AlexanderYastrebov/1brc)
* Documenting my experience and trying few things along the way....
* All tests on apple macbook M1 pro (2020) with 16 GB RAM.

# Solution 1:  Simple map with bufio.NewScanner. 
```
time go run main.go     
go run main.go  158.61s user 7.36s system 100% cpu 2:44.47 total
```

# Solution 2:  Simple map with bufio.NewReader. 
```
time go run main.go
go run main.go  181.55s user 6.01s system 99% cpu 3:08.80 total
```

# Solution 3: Worker pattern with explicit lock/unlock on map ... Performance twice as bad.
```
time go run main.go
Wrote to file 'output.txt'.
go run main.go  695.44s user 169.58s system 267% cpu 5:23.70 total
```

# Solution 4: Worker pattern with 4 workers and sync.Map.
```
time go run main.go
Wrote to file 'output.txt'.
go run main.go  582.49s user 201.67s system 320% cpu 4:04.51 total
```

# Solution 5: Batch processing with goroutine for each. Batch size: 1 million
```
time go run main.go 
Wrote to file 'output.txt'.
go run main.go  519.42s user 9.86s system 710% cpu 1:14.54 total
```

# Soluton 5.1: Batch processing but avoiding strings.split and using index.

```
time go run main.go 
Wrote to file 'output.txt'.
go run main.go  391.00s user 7.98s system 634% cpu 1:02.90 total
```

# Solution 6: Read file in 4 MB chunks. Consistently getting ~1 min. 
```
time go run main.go
Wrote to file 'output.txt'.
go run main.go  404.83s user 9.70s system 688% cpu 1:00.22 total
```

# Solution 7: (Baseline) Read file in 4 MB chunks and process data in using a buffered channel!
* Less system overload and ~20 sec.
* As a note, this solution works with buffered channel size of 1000. Otherwise, perf is ~50 sec.
* Steps (Step 1 and 3 are concurrent)
    1. Read data in 4 MB increments.
    2. Accumulates 1 million rows.
    3. Processes the batch of 1 million rows.
    4. Stores the result in a concurrent-safe sync.Map.
```
+------------------+     +------------------+     +------------------+
|   4 MB Chunk     | --> |   8 MB Chunk     | --> |  12 MB Chunk     | 
+------------------+     +------------------+     +------------------+
        |                        |                        |
        v                        v                        v
+--------------------------------------------------------------+
|                      Line Array (up to 1 million rows)       |
+--------------------------------------------------------------+
        |
        v
+--------------------------------------------------------------+
| Channel of 1 million rows processed into sync.Map            |
+--------------------------------------------------------------+
```

```
time go run main.go
Wrote to file 'output.txt'.
go run main.go  24.74s user 2.54s system 132% cpu 20.567 total
```

~~# Solution 8: Removed sync.Map. Make 2 channels. Multiple go routines writing to data channel (Fan out) and aggregating into result channel (map of station to temperature)~~

# Other solutions to explore:
1. Custom hashtables?
2. mmaps (memory mapped files with partial memory mapping based on page size)
