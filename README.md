# Fin-Tilt

Fin-Tilt is a command-line tool for managing and rebalancing your stock portfolio based on a configuration file that specifies a desired asset allocation and a CSV file containing your current portfolio values. It is based on the CSV file that Fidelity provides. The name Fin Tilt was inspired by the classic game [Full Tilt! Pinball](https://en.wikipedia.org/wiki/Full_Tilt!_Pinball).

## Installation

```sh
git clone https://github.com/ctil/fin-tilt
cd fin-tilt

# Build the tool
go build

# Print the usage message
./fin-tilt -h
```
## Configuration

Create a `config.yaml` file with your intended asset allocation. The config file has the following structure:

```yaml
stocks:
  - symbol: "VOO"
    target_percentage: 80.0
    description: "S&P 500 Index Fund"
  - symbol: "BND"
    target_percentage: 20.0
    description: "Total Bond Market Fund"
```

## Usage

### Rebalance

Rebalance your portfolio based on the current values provided in a CSV file.

```sh
./fin-tilt -config config.yaml rebalance portfolio.csv
```

Rebalance your portfolio while including an additional $5000 deposit.

```sh
./fin-tilt -config config.yaml rebalance -toDeposit 5000 portfolio.csv
```

The CSV file should have the following columns: `Symbol` and `Current Value`. If you download a CSV of your portfolio from Fidelity, it will have these columns.

### Deposit

Deposit a specified amount into your portfolio based on the target percentages defined in the configuration file.

```sh
./fin-tilt -config config.yaml deposit <amount>
```

Replace `<amount>` with the amount you want to deposit.

## License

This project is licensed under the MIT License.
