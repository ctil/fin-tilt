package main

import (
	"encoding/csv"
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
	DesiredPercentage float64
	Drift             float64
}

type Config struct {
	Stocks             []Stock `yaml:"stocks"`
	AvailableToDeposit int     `yaml:"available_to_deposit"`
}

type Stock struct {
	Symbol            string  `yaml:"symbol"`
	DesiredPercentage float64 `yaml:"desired_percentage"`
	Description       string  `yaml:"description"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to the config file")

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
	}

}

func rebalance(config *Config, args []string) {
	var portfolioCsv string
	flagSet := flag.NewFlagSet("rebalance", flag.ExitOnError)
	flagSet.StringVar(&portfolioCsv, "csv", "", "path to a Fidelity portfolio csv file")
	flagSet.Parse(args)

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
			if err == io.EOF {
				break
			}
			if parseErr, ok := err.(*csv.ParseError); ok && parseErr.Err == csv.ErrFieldCount {
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
		drift := currentPercentage - stock.DesiredPercentage
		data := SymbolData{
			Amount:            currentAmount,
			CurrentPercentage: currentPercentage,
			DesiredPercentage: stock.DesiredPercentage,
			Drift:             drift,
			AmountNeeded:      int(math.Floor(float64(total) * (-drift / 100))),
		}

		symbolData[stock.Symbol] = data
		needed := formatAmount(data.AmountNeeded)
		fmt.Printf("\n%s (%s)\n    Current: %05.2f%%, Desired: %05.2f%%, Drift: %+05.2f%%, Amount Needed: %s\n",
			stock.Symbol, stock.Description, data.CurrentPercentage, data.DesiredPercentage, data.Drift, needed)
	}

	fmt.Printf("\nTotal: %s\n", formatAmount(total))
}

func deposit(config *Config, args []string) {
	var amount string
	flagSet := flag.NewFlagSet("deposit", flag.ExitOnError)
	flagSet.StringVar(&amount, "amount", "", "amount to deposit")
	flagSet.Parse(args)
	fmt.Println("Amount to deposit: ", amount)
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

func formatAmount(amount int) string {
	amountStr := strconv.Itoa(amount)
	amountStr = amountStr[:len(amountStr)-2] + "." + amountStr[len(amountStr)-2:]
	return "$" + amountStr
}
