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
package main

func worker(ch chan int) {}

func main() {
    c := make(chan int)
    worker(c)
}
```

Expected result:

```
worker.ch (chan int) -> { main.go:6 make(chan int) }
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
Function parameter: worker.ch (chan int)

Possible channel allocation sites:
- main.go:6:10 make(chan int)
```

*(Note: This output corresponds to the [first version](01_mvp_implementation). The actual output of the latest [fourth version](04_static_analysis_implementation) is more detailed and includes more information about the actual channel flows.)*

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

The analysis state maps contextual SSA values (variables indexed by their parent function and paired with the executing goroutine) to sets of channel allocation sites:

```
State : ContextValue -> Set<ChannelAllocationSite>
```

Examples of analysis values:

- SSA values;
- channel-typed function parameters;
- phi nodes;
- function return values;
- call results.

Example:

```
State[main.x@main] = {AllocSite#1}
State[worker.ch@main.go:15] = {AllocSite#1, AllocSite#2}
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
[p] v := make(chan T) ⟹ p ∈ State[v]
```
*(where `p` is the allocation site program point)*

#### Assignment

```
[p] dst := src ⟹ State[src] ⊆ State[dst]
```

#### Phi node

```
[p] phi := φ(..., input, ...) ⟹ State[input] ⊆ State[phi]
```

#### Function call

```
[p] call func(..., arg, ...) ⟹ State[arg] ⊆ State[param]
```

#### Return

```
[p] return value ⟹ State[value] ⊆ State[func.return]
```

#### Call result

```
[p] result := call func(...) ⟹ State[func.return] ⊆ State[result]
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
- `interprocedural`, because information is propagated across function calls. *(Note: The analysis is monolithic and whole-program over the complete call graph starting from `main`. It does not compute modular procedure summaries).*

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

## Static Analysis (v4) (ACTUAL)

[Here](04_static_analysis_implementation) is the static analysis version.
[Here](04_static_analysis_implementation/FORMAL_SPECIFICATION.md) is the formal specification.

Added analysis of the traces from version 3 and finding precision and recall of the analyzer.

> **Supported Go Constructs:**
> *   Simple channel communication: Send (`ch <- v`) and Receive (`<-ch`)
> *   Channel closure: `close(ch)`
> *   Conditional branching: `if` / `else` (models both execution paths)
> *   Loops: `for` (models arbitrary loop iterations) and `for ... range` over channels
> *   Interprocedural data-flow: Passing channels as parameters to functions and goroutines

> **Known Limitations and Assumptions:**
> 1. **Channel-Isolated Trace Projection:** Traces are projected per channel, which discards cross-channel synchronization dependencies (e.g., waiting to read from channel A before writing to channel B). This over-approximation leads to False Positives (Deadlocks) in complex topologies like ring patterns.
> 2. **Unbuffered Channel Semantics:** All channels are treated as synchronous rendezvous (capacity = 0), regardless of their actual `make(chan T, N)` capacity in the source code.
> 3. **Bounded State Space Exploration:** The Bounded Model Checking performs BFS with a hard limit (`maxStates = 1000`) to prevent infinite loops, defensively reporting `Deadlock` if the bound is hit.
> 4. **Data-Flow Analysis Boundaries:** The constraint solver propagates channels via `Phi` nodes and `ChangeType` casts, but lacks full Interprocedural Pointer Alias Analysis (PTA). Channel passing through structs, maps, or closures may result in lost constraints.
> 5. **Control Flow Interruptions:** The AST-based trace builder ignores branches that do not contain channel operations. Consequently, early exits like `return`, `break`, or `continue` are not modeled. This can lead to False Negatives (reporting `Safe` instead of `Deadlock`), as the analyzer incorrectly assumes subsequent channel operations will always execute.
> 6. **Package-Level Variables & Init Functions:** The Call Graph traversal currently starts strictly from the `main()` function. Channels allocated globally at the package level or instantiated within `init()` blocks are not analyzed, causing any operations on them to be missed.

Current results:

```
=========================================================================================
ID     | Program                      | Expected   | Actual     | Verdict  | Time      
-----------------------------------------------------------------------------------------
GPH-01 | gph-deadlock                 | Deadlock   | Deadlock   | TP       | 65ms      
GPH-03 | gph-philo                    | Deadlock   | Deadlock   | TP       | 50ms      
GPH-04 | gph-primesieve               | Deadlock   | Deadlock   | TP       | 50ms      
GPH-05 | gph-primesieve-single        | Safe       | Deadlock   | FP       | 49ms      
GPH-06 | gph-barrier-safe             | Safe       | Safe       | TN       | 593ms     
GPH-07 | gph-barrier-deadlock         | Deadlock   | Deadlock   | TP       | 628ms     
DNG-01 | dng-local-deadlock           | Deadlock   | Deadlock   | TP       | 603ms     
DNG-02 | dng-local-deadlock-fixed     | Safe       | Safe       | TN       | 550ms     
DNG-07 | dng-simple                   | Deadlock   | Deadlock   | TP       | 550ms     
DNG-08 | dng-producer-consumer        | Safe       | Safe       | TN       | 578ms     
DNG-09 | dng-ring-pattern             | Safe       | Deadlock   | FP       | 587ms     
DNG-10 | dng-loop-variations          | Deadlock   | Deadlock   | TP       | 561ms     
DNG-11 | dng-factorial                | Safe       | Deadlock   | FP       | 571ms     
DNG-13 | dng-infinite-prime-sieve     | Safe       | Deadlock   | FP       | 683ms     
DNG-14 | dng-golang-blog-prime-sieve  | Safe       | Deadlock   | FP       | 1.167s    
STD-01 | std-unmatched-send           | Deadlock   | Deadlock   | TP       | 148ms     
STD-02 | std-unmatched-receive        | Deadlock   | Deadlock   | TP       | 145ms     
STD-03 | std-double-send              | Deadlock   | Deadlock   | TP       | 158ms     
STD-04 | std-double-receive           | Deadlock   | Deadlock   | TP       | 131ms     
STD-05 | std-loop-mismatch            | Deadlock   | Deadlock   | TP       | 58ms      
STD-06 | std-range-no-close           | Deadlock   | Deadlock   | TP       | 55ms      
STD-07 | std-early-return             | Deadlock   | Safe       | FN       | 54ms      
STD-08 | std-cond-deadlock            | Deadlock   | Deadlock   | TP       | 54ms      
STD-09 | std-worker-deadlock          | Deadlock   | Deadlock   | TP       | 58ms      
STD-10 | std-pipeline-deadlock        | Deadlock   | Deadlock   | TP       | 55ms      
STD-11 | std-circular-wait            | Deadlock   | Safe       | FN       | 59ms      
STD-12 | std-fork-join-safe           | Safe       | Safe       | TN       | 710ms     
STD-13 | std-fork-join-deadlock       | Deadlock   | Safe       | FN       | 717ms     
STD-14 | std-ping-pong-safe           | Safe       | Safe       | TN       | 664ms     
STD-15 | std-ping-pong-deadlock       | Deadlock   | Safe       | FN       | 607ms     
STD-16 | std-daisy-chain-safe         | Safe       | Safe       | TN       | 552ms     
STD-17 | std-daisy-chain-deadlock     | Deadlock   | Deadlock   | TP       | 549ms     
CUS-01 | cus-pipeline                 | Safe       | Safe       | TN       | 52ms      
CUS-02 | cus-deadlock                 | Deadlock   | Deadlock   | TP       | 546ms     
CUS-03 | cus-fanout-safe              | Safe       | Safe       | TN       | 572ms     
CUS-04 | cus-fanout-deadlock          | Deadlock   | Deadlock   | TP       | 535ms     
=========================================================================================
                               EVALUATION RESULTS                                        
=========================================================================================
Total Evaluated      : 36
True Positives (TP)  : 19
True Negatives (TN)  : 8
False Positives (FP) : 5
False Negatives (FN) : 4
-----------------------------------------------------------------------------------------
Precision            :  79.17%
Recall               :  82.61%
=========================================================================================
```