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

# Solution 3: Worker pattern ... Performance twice as bad.
```
time go run main.go
Wrote to file 'output.txt'.
go run main.go  695.44s user 169.58s system 267% cpu 5:23.70 total
```
