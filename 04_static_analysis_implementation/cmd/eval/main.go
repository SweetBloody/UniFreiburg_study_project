package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	loader "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/01_loader"
	ssa_builder "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/02_ssa"
	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/03_analysis"
	symbolic "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/04_symbolic"
	matcher "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/05_matcher"
)

type OracleEntry struct {
	ID        string
	Source    string
	Path      string
	Expected  string
	Supported string
	Reason    string
}

func main() {
	oraclePath := filepath.Join("benchmark", "oracle.csv")
	if len(os.Args) > 1 {
		oraclePath = os.Args[1]
	}

	entries, err := readOracle(oraclePath)
	if err != nil {
		log.Fatalf("Error reading oracle.csv: %v", err)
	}

	fmt.Println("=========================================================================================")
	fmt.Printf("%-6s | %-28s | %-10s | %-10s | %-8s | %-10s\n", "ID", "Program", "Expected", "Actual", "Verdict", "Time")
	fmt.Println("-----------------------------------------------------------------------------------------")

	var tp, fp, tn, fn int

	for _, entry := range entries {
		if strings.ToLower(entry.Supported) != "yes" {
			continue
		}

		targetPath := entry.Path
		if !strings.HasPrefix(targetPath, "./") && !strings.HasPrefix(targetPath, "/") {
			targetPath = "./" + targetPath
		}

		start := time.Now()
		actual, err := analyzeProgram(targetPath)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("%-6s | %-28s | %-10s | %-10s | %-8s | %-10v\n",
				entry.ID, filepath.Base(entry.Path), entry.Expected, "ERROR", "ERR", duration.Round(time.Millisecond))
			continue
		}

		verdict := getVerdict(entry.Expected, actual)
		switch verdict {
		case "TP":
			tp++
		case "FP":
			fp++
		case "TN":
			tn++
		case "FN":
			fn++
		}

		fmt.Printf("%-6s | %-28s | %-10s | %-10s | %-8s | %-10v\n",
			entry.ID, filepath.Base(entry.Path), entry.Expected, actual, verdict, duration.Round(time.Millisecond))
	}

	printSummary(tp, fp, tn, fn)
}

func readOracle(path string) ([]OracleEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var entries []OracleEntry
	for i, row := range records {
		if i == 0 || len(row) < 6 {
			continue // skip header or incomplete rows
		}
		entries = append(entries, OracleEntry{
			ID:        strings.TrimSpace(row[0]),
			Source:    strings.TrimSpace(row[1]),
			Path:      strings.TrimSpace(row[2]),
			Expected:  strings.TrimSpace(row[3]),
			Supported: strings.TrimSpace(row[4]),
			Reason:    strings.TrimSpace(row[5]),
		})
	}
	return entries, nil
}

func analyzeProgram(pkgPath string) (string, error) {
	pkgs, err := loader.LoadPackages(pkgPath)
	if err != nil {
		return "", err
	}

	prog, _ := ssa_builder.BuildSSA(pkgs)
	collector := analysis.NewCollector()
	collector.Collect(prog)

	analysis.Solve(collector.State, collector.Constraints)
	symBuilder := symbolic.NewBuilder(collector.State)
	symBuilder.Build(prog)

	projectedTraces := symBuilder.ProjectAll()
	deadlocks := matcher.DetectDeadlocks(projectedTraces)

	if len(deadlocks) > 0 {
		return "Deadlock", nil
	}
	return "Safe", nil
}

func getVerdict(expected, actual string) string {
	exp := strings.ToLower(strings.TrimSpace(expected))
	act := strings.ToLower(strings.TrimSpace(actual))

	if exp == "deadlock" && act == "deadlock" {
		return "TP"
	} else if exp == "deadlock" && act == "safe" {
		return "FN"
	} else if exp == "safe" && act == "deadlock" {
		return "FP"
	} else if exp == "safe" && act == "safe" {
		return "TN"
	}
	return "?"
}

func printSummary(tp, fp, tn, fn) {
	total := tp + fp + tn + fn

	precision := 0.0
	if (tp + fp) > 0 {
		precision = float64(tp) / float64(tp+fp) * 100.0
	} else if tp > 0 {
		precision = 100.0
	}

	recall := 0.0
	if (tp + fn) > 0 {
		recall = float64(tp) / float64(tp+fn) * 100.0
	}

	fmt.Println("=========================================================================================")
	fmt.Println("                               EVALUATION RESULTS                                        ")
	fmt.Println("=========================================================================================")
	fmt.Printf("Total Evaluated      : %d\n", total)
	fmt.Printf("True Positives (TP)  : %d\n", tp)
	fmt.Printf("True Negatives (TN)  : %d\n", tn)
	fmt.Printf("False Positives (FP) : %d\n", fp)
	fmt.Printf("False Negatives (FN) : %d\n", fn)
	fmt.Println("-----------------------------------------------------------------------------------------")
	fmt.Printf("Precision            : %6.2f%%\n", precision)
	fmt.Printf("Recall               : %6.2f%%\n", recall)
	fmt.Println("=========================================================================================")
}
