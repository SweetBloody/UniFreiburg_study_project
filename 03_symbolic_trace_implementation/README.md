# Documentation: Thread-Sensitive Channel Analysis (Version 3 — Symbolic Traces)

This version of the analyzer builds upon the previous stage. While the first stage (Goroutines Implementation) extracted a flat list of operations (`WRITE`, `READ`, `CLOSE`), this stage builds a symbolic trace by using `!` for `WRITE`, `?` for `READ` and `X` for `CLOSE`.

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
5. Prints the final result via `report.PrintSymbolicTraces()`.

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
    *(Note: Unlike Version 2, we no longer collect flat `READ/WRITE` operations here, as they are now extracted structurally by the AST Builder).*

### Stage 3: Solver Operation (`internal/03_analysis/solver.go`)
*   **`Solve(state, constraints)`**
    This is a "Work-List" algorithm. We have a set of data transfer rules (`Constraints`) and initial channel creation points (`State`). The solver takes these rules and pushes data from sources to targets in a loop. It runs until the system reaches a fixed point (until all variables know about all the channels they can point to).

### Stage 4: Structural Traversal (`internal/04_symbolic/builder.go`)
Once we have the solved `State`, we build the structural trace.
*   **`Build(prog)`**: Finds the `main` function and starts an AST traversal.
*   **`traverseASTNode(node)`**: Walks the Abstract Syntax Tree (AST) to preserve structural elements like `ast.ForStmt`, `ast.RangeStmt`, and `ast.IfStmt`.
*   **`ValueForExpr`**: When the builder encounters a channel operation (like `<-ch` in AST), it uses `fn.ValueForExpr()` to get the corresponding SSA value. It then checks the `State` to see which channel this operation belongs to.
*   **Inlining Goroutines**: When it encounters a function call or a goroutine spawn (`go func()`), it recursively traverses the called function and inlines its operations directly into the current execution path. This results in one massive unified tree representing the entire program execution.
*   **`ProjectAll(unifiedTrace)`**: A mathematical projection function. It extracts all unique `AllocSites` from the `State`, and for each channel, it filters the massive unified program trace down to only its relevant operations. It drops operations belonging to other channels and automatically prunes any empty loops or `if` branches. This yields a ready-to-print map of `Channel -> Trace`.

### Stage 5: Building the Final Domain (`internal/05_report/printer.go`)
*   **`PrintSymbolicTraces`**
    The printer takes the pre-calculated map of projected traces (`map[model.AllocSite][]model.TraceNode`) from the Builder and simply prints the resulting mathematical trace for each channel in a beautiful format.

## 3. Example output

```
Channel Allocation: /Users/sweetbloody/Documents/uni/UniFreiburg_study_project/03_symbolic_trace_implementation/testdata/workerpool/main.go:10:14 chan int
Trace: [ loop(3, [loop(*, [?])]), loop(5, [!]), X ]

Channel Allocation: /Users/sweetbloody/Documents/uni/UniFreiburg_study_project/03_symbolic_trace_implementation/testdata/workerpool/main.go:11:17 chan int
Trace: [ loop(3, [loop(*, [!])]), loop(5, [?]) ]
```
