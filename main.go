package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
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
	Stocks []Stock `yaml:"stocks"`
}

type Stock struct {
	Symbol           string  `yaml:"symbol"`
	TargetPercentage float64 `yaml:"target_percentage"`
	Description      string  `yaml:"description"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Config file that specifies a desired asset allocation")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: fin-tilt -config <config.yaml> <command> [<args>]\n")
		fmt.Println("Commands:")
		fmt.Println("  rebalance <portfolio.csv> [-toDeposit <amount>]  Rebalance portfolio based on current values in CSV file")
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
	var toDeposit int
	flagSet := flag.NewFlagSet("rebalance", flag.ExitOnError)
	flagSet.IntVar(&toDeposit, "toDeposit", 0, "Additional amount to deposit, in dollars")
	if len(args) < 1 {
		flag.Usage()
		return
	}
	portfolioCsv = args[0]
	flagSet.Parse(args[1:])

	// Convert to cents
	toDeposit *= 100

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
	total := toDeposit
	header, err := reader.Read()
	if err != nil {
		fmt.Println("Error reading header:", err)
		return
	}
	symbolIndex := slices.Index(header, "Symbol")
	amountIndex := slices.Index(header, "Current Value")
	if symbolIndex == -1 || amountIndex == -1 {
		fmt.Println("CSV file must have 'Symbol' and 'Current Value' columns")
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
		symbol := record[symbolIndex]

		if !symbolsToRebalance[symbol] {
			// Ignore any symbols that are not in the config
			continue
		}

		amount, err := amountToInt(record[amountIndex])
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

	fmt.Println("\n" + strings.Repeat("-", 60))
	if toDeposit > 0 {
		fmt.Printf("Total: %s (includes %s deposit)\n", formatAmount(total, true), formatAmount(toDeposit, true))
	} else {
		fmt.Printf("Total: %s\n", formatAmount(total, true))
	}
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

	totalPercentage := 0.0
	for _, stock := range config.Stocks {
		totalPercentage += stock.TargetPercentage
	}

	if math.Abs(totalPercentage-100.0) > 1e-9 {
		return nil, errors.New("target percentages do not add up to 100")
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
	if len(amountStr) < 3 {
		// Ensure at least 3 characters for slicing (e.g., "001" for 1 cent)
		amountStr = strings.Repeat("0", 3-len(amountStr)) + amountStr
	}
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
