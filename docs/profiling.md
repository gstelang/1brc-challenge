
# Profiling

* Install graphviz for flamegraphs, etc
```
brew install graphviz
```

* Command
```
go tool pprof cpu_profile.prof
go tool pprof mem_profile.prof

// you can view flamegraphs, etc in the web UI.
go tool pprof -http=:8080 cpu_profile.prof
go tool pprof -http=:8080 mem_profile.prof

// top for cpu
Showing nodes accounting for 19110ms, 87.38% of 21870ms total
Dropped 78 nodes (cum <= 109.35ms)
Showing top 10 nodes out of 88
      flat  flat%   sum%        cum   cum%
    9580ms 43.80% 43.80%     9580ms 43.80%  syscall.syscall
    3080ms 14.08% 57.89%     3080ms 14.08%  indexbytebody
    1110ms  5.08% 62.96%     5890ms 26.93%  bytes.genSplit
     920ms  4.21% 67.17%      920ms  4.21%  runtime.pthread_cond_wait
     890ms  4.07% 71.24%      890ms  4.07%  runtime.memmove
     820ms  3.75% 74.99%     2300ms 10.52%  runtime.scanobject
     780ms  3.57% 78.56%      780ms  3.57%  runtime.pthread_cond_signal
     730ms  3.34% 81.89%     4380ms 20.03%  bytes.Index
     620ms  2.83% 84.73%      620ms  2.83%  runtime.pthread_kill
     580ms  2.65% 87.38%     3660ms 16.74%  bytes.IndexByte (inline)
```
