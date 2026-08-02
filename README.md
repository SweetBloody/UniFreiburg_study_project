# UniFreiburg Study Project

## Static program analysis (Theory)

[Book](spa.pdf)

### Chapter 1 — Introduction

- Static analysis studies programs without executing them.
- Exact answers for non-trivial program behavior are generally undecidable.
- Therefore, our Go channel-flow analysis must be approximate.
- The analysis should conservatively report all channels that may flow into each channel-typed function parameter.
- This is necessary for later deadlock analysis: missing a real channel flow would be unsound, while extra flows only reduce precision.

### Chapter 2 — A Tiny Imperative Programming Language

- Static analysis requires a structured representation of the program, not just raw source code.
- The chapter introduces ASTs and Control Flow Graphs as basic representations for program analysis.
- For this project, we will analyze full Go programs using Go's existing analysis infrastructure instead of a toy language.
- SSA is useful because it normalizes complex Go code into simpler operations.
- Call graphs are needed to track how channel values flow between functions.

### Chapter 3 — Type Analysis

- The chapter shows how program properties can be inferred by generating and solving constraints.
- In Go type information is already available through the compiler infrastructure.
- We use this information to identify channel-typed parameters and channel-related values.
- The project adopts the constraint-based idea, but the constraints describe possible channel flows instead of types.
- A first implementation can conservatively propagate sets of channel allocation sites through assignments and function calls.

### Chapter 4 — Lattice Theory

- The chapter provides the mathematical foundation for approximate static analysis.
- For this project, possible channel values can be represented using a powerset lattice of channel allocation sites.
- The bottom element is the empty set, meaning that no channel source is known yet.
- The join operation is set union, used to merge possible channel flows from different assignments, branches, or calls.
- Monotone propagation over this lattice allows the analysis to compute a fixed point of possible channel flows.

### Chapter 5 — Dataflow Analysis

- The chapter explains how static analyses can be defined as dataflow problems over lattices.
- Our channel-flow analysis can be formulated as a forward may-analysis.
- Each relevant Go SSA value is mapped to a set of channel allocation sites that may flow into it.
- Transfer functions propagate channel sets through `make(chan)`, assignments, function calls, and returns.
- The final channel-flow information is computed as a fixed point, preferably using a work-list algorithm for efficiency.

## Go Static Analysis Infrastructure

### Analysing packages that may be useful

- [`go/packages`](tools_infrastructure/go_01_packages) is used as the entry point for loading Go packages with syntax trees, type information, and dependencies
- [`go/types`](tools_infrastructure/go_02_types) is used to identify channel-typed parameters, expressions, and values, so the analysis can focus only on channel-related data flow
- [`go/ssa`](tools_infrastructure/go_03_ssa) is the main intermediate representation for the analysis, because it exposes value flow through simpler instructions such as channel creation, calls, returns, and phi nodes
- [`go/callgraph`](tools_infrastructure/go_04_callgraph) is used to connect call sites with possible target functions, allowing channel-flow facts to be propagated from actual arguments to formal parameters
- `go/analysis` can be used later to package the analysis as a standard Go analyzer, but the first prototype can be implemented as a standalone CLI tool

### Suggested plan to implement the analysis

```
Go project / module -> go/packages -> go/types -> go/ssa -> go/callgraph -> analysis objects -> constraints -> fixed-point solver -> result
```

Step by step guide:

1. go/packages
   > Load target Go packages with syntax, types, and dependencies

2. go/types
   > Identify channel-typed parameters and channel-related values

3. go/ssa
   > Build SSA and inspect instructions:
   > `MakeChan`, `Call`, `Return`, `Phi`, `Parameter`

4. go/callgraph
   > Connect call sites with possible callee functions

5. Collect allocation sites
   > Each `make(chan T)` becomes a unique channel source

6. Build analysis state
   > State: SSAValue -> Set<ChannelAllocationSite>

7. Generate constraints
    ```
    make(chan)     -> State[v] contains AllocSite
    assignment     -> State[dst] ⊇ State[src]
    phi            -> State[phi] ⊇ State[input]
    call           -> State[param] ⊇ State[arg]
    return         -> State[func.return] ⊇ State[v]
    call result    -> State[v] ⊇ State[func.return]
    ```

8. Solve constraints
   > Use a work-list fixed-point solver

9. Output result
   > For each channel-typed function parameter, report possible channel allocation sites

## Formal Definition of the Channel-Flow Analysis

### Goal

The goal of the analysis is to compute, for each function parameter of channel type in a Go program, the set of channel allocation sites that may flow into this parameter at runtime.

A `channel allocation site` is a program location where a new channel is created using:

```go
make(chan T)
```

Example:

```go
func worker(ch chan int) {}

func main() {
    c := make(chan int)
    worker(c)
}
```

Expected result:

```
worker.ch -> { main.go:5 make(chan int) }
```

### Input

The input of the analysis is a Go package or module loaded through the Go analysis infrastructure.

The analysis uses:

- `go/packages` to load Go packages, syntax trees, type information, imports, and dependencies;
- `go/types` to identify channel-typed parameters and values;
- `go/ssa` to inspect value-flow instructions;
- `go/callgraph` to connect call sites with possible callee functions.

### Output

The output is a mapping:

```
ChannelParameter -> Set<ChannelAllocationSite>
```

For each function parameter whose type is `chan T`, `<-chan T`, or `chan<- T`, the analysis reports all channel allocation sites that may flow into this parameter.

Example output:

```
Function parameter: worker.ch chan int

Possible channel allocation sites:
- main.go:5:10 make(chan int)
- main.go:6:10 make(chan int)
```

### Abstract domain

The abstract domain is the powerset of channel allocation sites:

```
ChannelSet = P(ChannelAllocationSites)
```

Each element of this domain is a set of possible channel allocation sites.

Examples:

```
{}
{AllocSite#1}
{AllocSite#2}
{AllocSite#1, AllocSite#2}
```

### Abstract state

The analysis state maps SSA values and other analysis entities to sets of channel allocation sites:

```
State : Value -> Set<ChannelAllocationSite>
```

Examples of analysis values:

- SSA values;
- channel-typed function parameters;
- phi nodes;
- function return values;
- call results.

Example:

```
State[x] = {AllocSite#1}
State[worker.ch] = {AllocSite#1, AllocSite#2}
```

### Constraint form

The analysis generates flow constraints of the form:

```
State[target] ⊇ State[source]
```

This means that every channel allocation site that may flow into `source` must also be considered as possibly flowing into `target`.

In implementation terms:

```
State[target] = State[target] ∪ State[source]
```

### Core constraints

#### Channel creation

```
make(chan T) -> State[v] contains AllocSite
```

#### Assignment

```
State[dst] ⊇ State[src]
```

#### Phi node

```
State[phi] ⊇ State[input]
```

#### Function call

```
State[param] ⊇ State[arg]
```

#### Return

```
State[func.return] ⊇ State[value]
```

#### Call result

```
State[result] ⊇ State[func.return]
```


### Solver

The constraints are solved using a `work-list fixed-point algorithm`.

The algorithm starts with empty channel sets for all values. Channel allocation sites are added as initial facts. Then the solver repeatedly propagates channel sets through constraints until no set changes anymore.

Since the number of channel allocation sites and SSA values in a finite Go program is finite, and each set can only grow, the algorithm eventually reaches a fixed point.

### Analysis properties

The analysis is:

- `static`, because it analyzes the program without executing it
- `conservative`, because it over-approximates possible channel flows
- `may-analysis`, because it computes which channels may flow into each parameter
- `forward`, because channel-flow information is propagated from channel creation sites to later uses
- `interprocedural`, because information is propagated across function calls

The analysis may report extra possible channel flows, but it should not miss real channel flows within the supported language subset.

## MVP Implementation (v1)

[Here](01_mvp_implementation) is a first so-called MVP version of the implementation.

## Goroutine-sensitive implementation (v2)

[Here](02_goroutines_implementation) is the goroutine-sensitive version of the implementation.

## Symbolic-trace implementation (v3)

[Here](03_symbolic_trace_implementation) is the symbolic trace version of the implementation.

### Idea

To represent channel operations symbolically by using `!` for `WRITE`, `?` for `READ` and `X` for `CLOSE`. And then use such a cymbolic trace for the future channel analysis.

Example:
```go
ch <- 1 
for i<5 {
   ch <- 2 
}
<-ch  
```
will be represented as:

```
[!, loop(5, [!]), ?]
```

### What changed

As we don't have any information about the control flow in our `SSA`, we need to use `AST` which helps us to see loops and branches.

To get the best of both `SSA` and `AST`, we use a hybrid approach:

> We still use `SSA`-builder but with a flag `ssa.GlobalDebug` which allows to save the debug information which helps to link the exact AST nodes with their corresponding SSA values.

## Static Analysis (v4)

[Here](04_static_analysis_implementation) is the static analysis version.

Added analysis of the traces from version 3 and finding precision and recall of the analyzer.

Current results:

```
=========================================================================================
ID     | Program                      | Expected   | Actual     | Verdict  | Time      
-----------------------------------------------------------------------------------------
GPH-01 | gph-deadlock                 | Deadlock   | Deadlock   | TP       | 44ms      
GPH-03 | gph-philo                    | Deadlock   | Deadlock   | TP       | 46ms      
GPH-04 | gph-primesieve               | Deadlock   | Deadlock   | TP       | 47ms      
GPH-05 | gph-primesieve-single        | Safe       | Deadlock   | FP       | 46ms      
GPH-06 | gph-barrier-safe             | Safe       | Safe       | TN       | 682ms     
GPH-07 | gph-barrier-deadlock         | Deadlock   | Safe       | FN       | 632ms     
DNG-01 | dng-local-deadlock           | Deadlock   | Deadlock   | TP       | 618ms     
DNG-02 | dng-local-deadlock-fixed     | Safe       | Deadlock   | FP       | 609ms     
DNG-07 | dng-simple                   | Deadlock   | Deadlock   | TP       | 583ms     
DNG-08 | dng-producer-consumer        | Safe       | Safe       | TN       | 594ms     
DNG-09 | dng-ring-pattern             | Safe       | Deadlock   | FP       | 552ms     
DNG-10 | dng-loop-variations          | Deadlock   | Deadlock   | TP       | 572ms     
DNG-11 | dng-factorial                | Safe       | Deadlock   | FP       | 572ms     
DNG-13 | dng-infinite-prime-sieve     | Safe       | Deadlock   | FP       | 594ms     
DNG-14 | dng-golang-blog-prime-sieve  | Safe       | Deadlock   | FP       | 591ms     
STD-01 | std-unmatched-send           | Deadlock   | Deadlock   | TP       | 60ms      
STD-02 | std-unmatched-receive        | Deadlock   | Deadlock   | TP       | 85ms      
STD-03 | std-double-send              | Deadlock   | Deadlock   | TP       | 122ms     
STD-04 | std-double-receive           | Deadlock   | Deadlock   | TP       | 83ms      
STD-05 | std-loop-mismatch            | Deadlock   | Deadlock   | TP       | 85ms      
STD-06 | std-range-no-close           | Deadlock   | Deadlock   | TP       | 74ms      
STD-07 | std-early-return             | Deadlock   | Safe       | FN       | 70ms      
STD-08 | std-cond-deadlock            | Deadlock   | Deadlock   | TP       | 76ms      
STD-09 | std-worker-deadlock          | Deadlock   | Deadlock   | TP       | 77ms      
STD-10 | std-pipeline-deadlock        | Deadlock   | Deadlock   | TP       | 74ms      
STD-11 | std-circular-wait            | Deadlock   | Safe       | FN       | 75ms      
STD-12 | std-fork-join-safe           | Safe       | Deadlock   | FP       | 660ms     
STD-13 | std-fork-join-deadlock       | Deadlock   | Deadlock   | TP       | 846ms     
STD-14 | std-ping-pong-safe           | Safe       | Safe       | TN       | 650ms     
STD-15 | std-ping-pong-deadlock       | Deadlock   | Safe       | FN       | 581ms     
STD-16 | std-daisy-chain-safe         | Safe       | Deadlock   | FP       | 541ms     
STD-17 | std-daisy-chain-deadlock     | Deadlock   | Deadlock   | TP       | 565ms     
CUS-01 | cus-pipeline                 | Safe       | Safe       | TN       | 56ms      
CUS-02 | cus-deadlock                 | Deadlock   | Deadlock   | TP       | 581ms     
CUS-03 | cus-fanout-safe              | Safe       | Deadlock   | FP       | 620ms     
CUS-04 | cus-fanout-deadlock          | Deadlock   | Deadlock   | TP       | 553ms     
=========================================================================================
                               EVALUATION RESULTS                                        
=========================================================================================
Total Evaluated      : 36
True Positives (TP)  : 19
True Negatives (TN)  : 4
False Positives (FP) : 9
False Negatives (FN) : 4
-----------------------------------------------------------------------------------------
Precision            :  67.86%
Recall               :  82.61%
=========================================================================================
```