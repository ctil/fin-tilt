package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type TestDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Command     string         `json:"command"`
	ConfigFile  string         `json:"config_file"`
	Input       TestInput      `json:"input"`
	Expected    ExpectedResult `json:"expected"`
	Tolerance   float64        `json:"tolerance"`
}

type TestInput struct {
	CSVFile       string `json:"csv_file"`
	DepositAmount int    `json:"deposit_amount"`
}

type ExpectedResult struct {
	Total   int                       `json:"total"`
	Symbols map[string]ExpectedSymbol `json:"symbols"`
}

type ExpectedSymbol struct {
	Amount            int     `json:"amount"`
	CurrentPercentage float64 `json:"current_percentage"`
	Drift             float64 `json:"drift"`
	AmountNeeded      int     `json:"amount_needed"`
}

func TestRebalanceFromDefinitions(t *testing.T) {
	testDataDir := "tests"
	definitionsDir := filepath.Join(testDataDir, "definitions")

	entries, err := os.ReadDir(definitionsDir)
	if err != nil {
		t.Fatalf("Failed to read test definitions directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		defPath := filepath.Join(definitionsDir, entry.Name())
		defBytes, err := os.ReadFile(defPath)
		if err != nil {
			t.Fatalf("Failed to read test definition %s: %v", entry.Name(), err)
		}

		var def TestDefinition
		if err := json.Unmarshal(defBytes, &def); err != nil {
			t.Fatalf("Failed to parse test definition %s: %v", entry.Name(), err)
		}

		if def.Command != "rebalance" {
			continue
		}

		t.Run(def.Name, func(t *testing.T) {
			configPath := filepath.Join(testDataDir, def.ConfigFile)
			config, err := parseConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			csvPath := filepath.Join(testDataDir, def.Input.CSVFile)
			csvFile, err := os.Open(csvPath)
			if err != nil {
				t.Fatalf("Failed to open CSV file: %v", err)
			}
			defer csvFile.Close()

			result, err := rebalanceCalc(config, csvFile, def.Input.DepositAmount)
			if err != nil {
				t.Fatalf("rebalanceCalc failed: %v", err)
			}

			if result.Total != def.Expected.Total {
				t.Errorf("Total mismatch: got %d, expected %d", result.Total, def.Expected.Total)
			}

			for symbol, expected := range def.Expected.Symbols {
				actual, ok := result.Symbols[symbol]
				if !ok {
					t.Errorf("Symbol %s not found in result", symbol)
					continue
				}

				if actual.Amount != expected.Amount {
					t.Errorf("Symbol %s: Amount mismatch: got %d, expected %d", symbol, actual.Amount, expected.Amount)
				}

				if !floatEqual(actual.CurrentPercentage, expected.CurrentPercentage, def.Tolerance) {
					t.Errorf("Symbol %s: CurrentPercentage mismatch: got %f, expected %f", symbol, actual.CurrentPercentage, expected.CurrentPercentage)
				}

				if !floatEqual(actual.Drift, expected.Drift, def.Tolerance) {
					t.Errorf("Symbol %s: Drift mismatch: got %f, expected %f", symbol, actual.Drift, expected.Drift)
				}

				if actual.AmountNeeded != expected.AmountNeeded {
					t.Errorf("Symbol %s: AmountNeeded mismatch: got %d, expected %d", symbol, actual.AmountNeeded, expected.AmountNeeded)
				}
			}
		})
	}
}

func floatEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
