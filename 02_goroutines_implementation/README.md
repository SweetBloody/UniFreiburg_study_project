# Documentation: Thread-Sensitive Channel Analysis (Version 2 — Goroutines)

This version of the analyzer is able to distinguish between function calls occurring in different goroutines. The main goal: to determine which goroutines read from a channel, and which write to it or close it.

---

## 1. Main Data Structures and Their Fields

All key structures are located in the `internal/model` and `internal/03_analysis` packages.

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

### `OpType` and `ChanOp`
Describe the actions that happen with the channel.
*   **`OpType`**: The type of action (`READ`, `WRITE`, `CLOSE`).
*   **`ChanOp`**: A structure storing:
    *   `Type`: the type of operation.
    *   `ChannelVar` (`ContextValue`): on which contextual variable the action was performed.
    *   `Position`: where in the code this happened.

### `State`
The knowledge map of the system: `map[ContextValue]map[AllocSite]struct{}`.
*   Answers the question: "Which actual channels (`AllocSite`) can this variable point to in this goroutine (`ContextValue`)?".

### `Collector`
The main "gatherer" of information during code traversal.
*   **`State`**: The knowledge base about where channels are created.
*   **`Constraints`**: The collected data transfer rules.
*   **`Operations`**: The collected flat list of all `READ`/`WRITE`/`CLOSE` operations.
*   **`visited`**: List of visited functions (to prevent infinite recursion).

---

## 2. Work Logic and Function Algorithms

Code analysis occurs in several stages:

### Stage 1: Start of Analysis (`cmd/chanflow/main.go`)
1. The program reads the source code and builds SSA (internal code representation as basic blocks and instructions).
2. Calls `Collector.Collect()` to gather all information about channels and goroutines.
3. Passes the collected data to the Solver (`Solve()`).
4. Prints the final result via `PrintResults()`.

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
    *   If it sees a read, write (`<-`) or `close()`, it saves this operation in `Operations`, noting which goroutine did it.
    *   If it sees a control flow branch node (`*ssa.Phi`), which occurs when a channel is selected via an `if/else` condition, it generates rules to merge data flows from all possible branches into the resulting variable.

### Stage 3: Solver Operation (`internal/03_analysis/solver.go`)
*   **`Solve(state, constraints)`**
    This is a "Work-List" algorithm. We have a set of data transfer rules (`Constraints`) and initial channel creation points (`State`). The solver takes these rules and pushes data from sources to targets in a loop. It runs until the system reaches a fixed point (until all variables know about all the channels they can point to).

### Stage 4: Building the Final Domain (`internal/04_report/printer.go`)
*   **`PrintResults(collector)`**
    The solver has finished, and now we know exactly which actual channel each contextual variable points to. The printer takes our flat list of `Operations` and sorts it out:
    1. Takes an operation (e.g., `WRITE` to variable `X`).
    2. Asks the Solver: "Which actual channel does `X` point to?".
    3. Puts this operation into the final structure by hierarchy: `Actual Channel ➔ Goroutine ➔ List of actions`.
    4. Prints a beautiful report.

## 3. Example output

```
Channel Allocation: /Users/sweetbloody/Documents/uni/UniFreiburg_study_project/02_goroutines_implementation/testdata/goroutines/main.go:13:11 chan int
- Goroutine '/Users/sweetbloody/Documents/uni/UniFreiburg_study_project/02_goroutines_implementation/testdata/goroutines/main.go:15':
    - READ
- Goroutine '/Users/sweetbloody/Documents/uni/UniFreiburg_study_project/02_goroutines_implementation/testdata/goroutines/main.go:16':
    - CLOSE
    - WRITE
```
