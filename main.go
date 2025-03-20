package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type SymbolData struct {
	Amount            int
	AmountNeeded      int
	CurrentPercentage float64
	TargetPercentage  float64
	Drift             float64
}

type Config struct {
	Stocks             []Stock `yaml:"stocks"`
	AvailableToDeposit int     `yaml:"available_to_deposit"`
}

type Stock struct {
	Symbol           string  `yaml:"symbol"`
	TargetPercentage float64 `yaml:"target_percentage"`
	Description      string  `yaml:"description"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to the config file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: fin-tilt -config <config.yaml> <command> [<args>]\n")
		fmt.Println("Commands:")
		fmt.Println("  rebalance <portfolio.csv>  Rebalance the portfolio based on the given CSV file")
		fmt.Println("  deposit <amount>           Deposit the specified amount")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	subCmd := flag.Arg(0)
	subCmdArgs := flag.Args()[1:]

	config, err := parseConfig(configPath)
	if err != nil {
		fmt.Println("Error parsing config:", err)
		os.Exit(1)
	}

	switch subCmd {
	case "rebalance":
		rebalance(config, subCmdArgs)
	case "deposit":
		deposit(config, subCmdArgs)
	default:
		fmt.Println("Unknown command:", subCmd)
		flag.Usage()
		os.Exit(1)
	}
}

func rebalance(config *Config, args []string) {
	var portfolioCsv string
	flagSet := flag.NewFlagSet("rebalance", flag.ExitOnError)
	if len(args) < 1 {
		flag.Usage()
		return
	}
	portfolioCsv = args[0]
	flagSet.Parse(args[1:])

	file, err := os.Open(portfolioCsv)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	symbolsToRebalance := make(map[string]bool)
	for _, stock := range config.Stocks {
		symbolsToRebalance[stock.Symbol] = true
	}
	reader := csv.NewReader(file)
	amountsBySymbol := make(map[string]int)
	total := 0
	// Skip the first line (header)
	if _, err := reader.Read(); err != nil {
		fmt.Println("Error reading header:", err)
		return
	}
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, csv.ErrFieldCount) {
				// The fidelity csv has some malformed lines at the end
				continue
			}
			fmt.Println("Error:", err)
			return
		}
		symbol := record[2]

		if !symbolsToRebalance[symbol] {
			// Ignore any symbols that are not in the config
			continue
		}

		amount, err := amountToInt(record[7])
		total += amount
		amountsBySymbol[symbol] += amount
		if err != nil {
			fmt.Println("Error parsing amount:", err)
			return
		}
	}
	symbolData := make(map[string]SymbolData)
	for _, stock := range config.Stocks {
		currentAmount := amountsBySymbol[stock.Symbol]
		currentPercentage := (float64(currentAmount) / float64(total)) * 100
		drift := currentPercentage - stock.TargetPercentage
		data := SymbolData{
			Amount:            currentAmount,
			CurrentPercentage: currentPercentage,
			TargetPercentage:  stock.TargetPercentage,
			Drift:             drift,
			AmountNeeded:      int(math.Round(float64(total) * (-drift / 100))),
		}

		symbolData[stock.Symbol] = data
		needed := formatAmount(data.AmountNeeded, false)
		if data.AmountNeeded > 0 {
			needed = green("+" + needed)
		} else {
			needed = red(needed)
		}
		driftStr := fmt.Sprintf("%.2f%%", data.Drift)
		if data.Drift > 0 {
			driftStr = green("+" + driftStr)
		} else {
			driftStr = red(driftStr)
		}
		fmt.Println("\n" + strings.Repeat("-", 60))
		fmt.Printf("%s - %.2f%% (%s)\n", stock.Symbol, data.CurrentPercentage, driftStr)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%s\n", stock.Description)
		fmt.Printf("Needed: %s\n", needed)
		fmt.Printf("Current Total: %s\n", formatAmount(data.Amount, true))
	}

	fmt.Printf("\nTotal: %s\n", formatAmount(total, true))
}

func green(str string) string {
	return "\033[32m" + str + "\033[0m"
}

func red(str string) string {
	return "\033[31m" + str + "\033[0m"
}

func deposit(config *Config, args []string) {
	var amount int
	flagSet := flag.NewFlagSet("deposit", flag.ExitOnError)
	if len(args) < 1 {
		flag.Usage()
		return
	}
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("Error parsing amount:", err)
		return
	}
	flagSet.Parse(args[1:])

	// Convert amount to cents
	amount *= 100
	total := 0

	for _, stock := range config.Stocks {
		amountToDeposit := int(math.Floor(float64(amount) * (stock.TargetPercentage / 100)))
		total += amountToDeposit
		fmt.Printf("%s: %s\n", stock.Symbol, formatAmount(amountToDeposit, false))
	}
}

func parseConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func amountToInt(amount string) (int, error) {
	amount = strings.TrimPrefix(amount, "$")
	amount = strings.ReplaceAll(amount, ".", "")
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		return 0, err
	}
	return amountInt, nil
}

func formatAmount(amount int, includeCommas bool) string {
	amountStr := strconv.Itoa(amount)
	dollars := amountStr[:len(amountStr)-2]
	cents := amountStr[len(amountStr)-2:]
	if includeCommas {
		for i := len(dollars) - 3; i > 0; i -= 3 {
			dollars = dollars[:i] + "," + dollars[i:]
		}
	}
	amountStr = dollars + "." + cents
	if amount < 0 {
		return amountStr[:1] + "$" + amountStr[1:]
	}
	return "$" + amountStr
}
