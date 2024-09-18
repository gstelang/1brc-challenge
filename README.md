# 1brc-challenge
* Documenting my experience and trying few things along the way....

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

# Solution 6: (TODO) Read file in chunks


# Other solutions to explore:
1. Custom hashtables.
2. mmaps