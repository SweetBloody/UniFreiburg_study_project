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

## MVP Implementation

[Here](mvp_implementation) is a first so-called MVP version of the implementation.