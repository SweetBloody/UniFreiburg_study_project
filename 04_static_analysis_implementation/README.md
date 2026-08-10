# Documentation: Static Analysis (Version 4 — Analysis of Traces)

This version of the analyzer builds upon the previous stage. While the first stage (Goroutines Implementation) extracted a flat list of operations (`WRITE`, `READ`, `CLOSE`), this stage builds a symbolic trace by using `!` for `WRITE`, `?` for `READ` and `X` for `CLOSE`.

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

> **Formal Mathematical Specification**: For a complete formal definition of the symbolic trace domain, transfer functions, operational semantics, and deadlock detection conditions, see **[FORMAL_SPECIFICATION.md](FORMAL_SPECIFICATION.md)**.

---

## 1. Main Data Structures and Their Fields

All key structures are located in the `internal/model`, `internal/03_analysis`, and `internal/04_symbolic` packages.

### `ValueID` (string)
A simple string identifier of a variable from the source code. The analyzer gives each variable a unique name (for example, `main.ch` or `worker.t1`).

### `GoroutineID` (string)
Identifier of a specific thread (goroutine). 
*   For the main goroutine, it is `"main"`.
*   For new goroutines, it is generated at the call site, e.g. `"main.go:15"` (the file and line number where `go func()` was written).

### `ContextValue`
The most important structure. It combines a variable and a goroutine.
*   **`Value`** (`ValueID`): Which variable.
*   **`Goroutine`** (`GoroutineID`): Inside which goroutine it currently exists.
*   *Why is it needed?* To distinguish the variable `ch` inside the `worker` function called from Goroutine A, from the same variable `ch` inside the `worker` function called from Goroutine B.

### `AllocSite`
The place in the code where the channel was created (where `make(chan T)` was called).
*   **`ID`**: Unique sequential number (1, 2, 3...).
*   **`Position`**: File and line of creation (e.g. `main.go:10`).
*   **`Type`**: The type of the created channel (e.g. `chan int`).

### `Constraint`
A data transfer rule (constraint).
*   **`Source`** (`ContextValue`): The source variable (where the channel was passed from).
*   **`Target`** (`ContextValue`): The target variable (where it was passed to).
*   *Why is it needed?* When we call `worker(c)`, a Constraint is created stating: "Everything that was in variable `c` of the caller flows into the parameter of the `worker` function".

### `OpType`
Describes the actions that happen with the channel: `READ`, `WRITE`, `CLOSE`.

### `State`
The knowledge map of the system: `map[ContextValue]map[AllocSite]struct{}`.
*   Answers the question: "Which actual channels (`AllocSite`) can this variable point to in this goroutine (`ContextValue`)?".

### `Collector`
The main "gatherer" of information during code traversal.
*   **`State`**: The knowledge base about where channels are created.
*   **`Constraints`**: The collected data transfer rules.
*   **`visited`**: List of visited functions (to prevent infinite recursion).

### `TraceNode` (interface)
The base interface for all elements in our symbolic trace.
*   **`OpNode`**: A leaf node representing a single channel operation (`!` for WRITE, `?` for READ, `X` for CLOSE). It also holds the `ContextValue` of the channel.
*   **`LoopNode`**: Represents a loop, formatted as `loop(bounds, [body])`.
*   **`IfNode`**: Represents conditional branching, formatted as `if(cond, [then], [else])`.
*   **`RangeNode`**: Represents a `for ... range` loop over a channel, formatted as `range(?*, [body])` if a body exists, or just `?*` if it's empty (consumes all values until the channel is closed).

### `Builder`
The main constructor of the structural trace.
*   **`State`**: Uses the solved data flow state to filter operations.
*   **`visited`**: List of visited functions to prevent infinite recursion.

---

## 2. Work Logic and Function Algorithms

Code analysis occurs in several stages:

### Stage 1: Start of Analysis (`cmd/chanflow/main.go`)
1. The program reads the source code and builds SSA (internal code representation as basic blocks and instructions). *Addition:* We now run the SSA builder with `ssa.GlobalDebug` enabled to preserve mapping to the AST.
2. Calls `Collector.Collect()` to gather all information about channels and goroutines.
3. Passes the collected data to the Solver (`Solve()`).
4. Calls `symbolic.Builder.Build()` to extract the unified structural trace, then `ProjectAll()` to slice it by channel.
5. Calls `matcher.DetectDeadlocks(projectedTraces)` to perform BFS state-space exploration and prints the final result via `report.PrintReport(projectedTraces, deadlocks)`.

### Stage 2: Graph Traversal and Fact Gathering (`internal/03_analysis/collector.go`)
*   **`Collect(prog)`**
    Finds the `main` function and starts a recursive traversal from it (the `traverse` function), passing the initial context — the `"main"` goroutine.
*   **`traverse(node, gID)`**
    The traveler function. It enters the current function (knowing the current `gID`) and does the following things:
    1. Examines all instructions inside (calling `processInstruction`).
    2. Looks at all outgoing calls from this function. If it sees a new goroutine launch (`go worker()`), it creates a new `nextGID`. If it's a regular call — it keeps the old `nextGID`.
    3. Builds data flow paths:
       - **Incoming (Arguments)**: calls the `matchArgumentConstraints` function, which links passed channels to function parameters.
       - **Outgoing (Return values)**: calls the `matchReturnConstraints` function. It scans the function for `return` statements and links returned channels to the variable in the calling goroutine. Supports even multiple returns (when a function returns a tuple of channels and they are unpacked via the `*ssa.Extract` instruction).
    4. Recursively "dives" further (calls itself for the next function).
*   **`processInstruction(instr, gID)`**
    Analyzes every line:
    *   If it sees `make(chan)`, it records this channel in the `State` base.
    *   If it sees a control flow branch node (`*ssa.Phi`), which occurs when a channel is selected via an `if/else` condition, it generates rules to merge data flows from all possible branches into the resulting variable.

### Stage 3: Solver Operation (`internal/03_analysis/solver.go`)
*   **`Solve(state, constraints)`**
    This is a "Work-List" algorithm. We have a set of data transfer rules (`Constraints`) and initial channel creation points (`State`). The solver takes these rules and pushes data from sources to targets in a loop. It runs until the system reaches a fixed point (until all variables know about all the channels they can point to).

### Stage 4: Structural Traversal (`internal/04_symbolic/builder.go`)
Once we have the solved `State`, we build the structural trace.
*   **`Build(prog)`**: Finds the `main` function and starts an AST traversal.
*   **`traverseASTNode(node)`**: Walks the Abstract Syntax Tree (AST) to preserve structural elements like `ast.ForStmt`, `ast.RangeStmt`, and `ast.IfStmt`.
*   **`ValueForExpr`**: When the builder encounters a channel operation (like `<-ch` in AST), it uses `fn.ValueForExpr()` to get the corresponding SSA value. It then checks the `State` to see which channel this operation belongs to.
*   **Goroutine Separation**: When it encounters a function call, it recursively traverses and inlines it. However, when it encounters a goroutine spawn (`go func()`), it creates a separate, independent trace for the new goroutine. The Builder stores all these traces internally as `map[GoroutineID][]TraceNode`.
*   **`ProjectAll()`**: A mathematical projection function. It extracts all unique `AllocSites` from the `State`, and for each channel, it filters the traces of all goroutines down to only their relevant operations. It drops operations belonging to other channels and automatically prunes any empty loops or `if` branches. This yields a ready-to-print map of `Channel -> Goroutine -> Trace`.

### Stage 5: Deadlock Detection via Symbolic Execution (`internal/05_matcher`)
*   **`MachineState` & State Space Exploration (BFS)**
    Instead of relying on heuristic pattern matching or simple operation counting, the matcher implements an operational semantics machine for each channel's projected traces. A state (`MachineState`) consists of the remaining trace for each active goroutine, the channel's `Closed` boolean flag, and a `Panicked` boolean flag (set if a goroutine performs an invalid operation like writing to a closed channel).
*   **`NextStates()` & Normalization**
    The engine computes valid transitions from any given state:
    *   **Control Flow Normalization**: If a goroutine's head is an `if` branch, two parallel states are created (`Then` and `Else`). If it is a `loop(*)`, two states are created (exit loop immediately, or execute one iteration and repeat).
    *   **Communication Transitions (`COMM`)**: Synchronous rendezvous where a sender (`!`) and a receiver (`?` or `?*`) exchange data simultaneously, advancing both traces.
    *   **Channel Operations**: Handling close (`X`) and non-blocking reads/ranges on closed channels.
*   **Deadlock Detection**
    The algorithm performs a Breadth-First Search (BFS) over the reachable state space, hashing states (`HashState`) into a `visited` set to prevent cycles and using Bounded Model Checking heuristics (`maxStates`). If a terminal state (`NextStates() == ∅`) is reached where at least one goroutine still has pending operations, a deadlock (or May-Deadlock) is reported.

### Stage 6: Reporting and Tracing (`internal/06_report/printer.go`)
*   **`PrintReport` & `PrintSymbolicTraces`**
    The printer takes the pre-calculated map of projected traces (`map[model.AllocSite]map[model.GoroutineID][]model.TraceNode`) from the Builder, prints the resulting mathematical trace grouped by each goroutine, and outputs the final deadlock analysis verdict calculated by `matcher.DetectDeadlocks`.

## 3. Example output

```text
Channel Allocation: /Users/sweetbloody/Documents/uni/UniFreiburg_study_project/04_static_analysis_implementation/testdata/pipeline/main.go:25:12 chan int
  Goroutine main.go:28: [ loop(*, [!]), X ]
  Goroutine main.go:29: [ ?* ]

Channel Allocation: /Users/sweetbloody/Documents/uni/UniFreiburg_study_project/04_static_analysis_implementation/testdata/pipeline/main.go:24:12 chan int
  Goroutine main.go:27: [ loop(5, [!]), X ]
  Goroutine main.go:28: [ ?* ]

No deadlocks detected!
```