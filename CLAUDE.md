# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fin-Tilt is a command-line tool for rebalancing stock portfolios. It calculates rebalancing recommendations based on:
- A YAML configuration file defining target asset allocation percentages
- A CSV file containing current portfolio holdings (Fidelity format or similar)

The tool operates in two modes:
1. **rebalance**: Analyzes current portfolio and recommends buys/sells to reach target allocation
2. **deposit**: Calculates how to allocate a new deposit across assets per target percentages

## Build and Run Commands

```sh
# Build the binary
go build

# Run with usage info
./fin-tilt -h

# Rebalance portfolio
./fin-tilt -config config.yaml rebalance portfolio.csv

# Rebalance with additional deposit
./fin-tilt -config config.yaml rebalance -toDeposit 5000 portfolio.csv

# Calculate deposit allocation
./fin-tilt -config config.yaml deposit 5000
```

## Configuration Files

The repository includes example configuration files:
- `config.yaml`: Default configuration
- `bonds_10.yaml`: Configuration with 10% bonds
- `no_bonds.yaml`: Configuration without bonds

Config structure requires:
- `stocks` array with `symbol`, `target_percentage`, and `description`
- Target percentages must sum to exactly 100.0 (validated in `parseConfig()`)

## Code Architecture

### Single-File Structure

All code is in `main.go` (~273 lines). Key components:

**Data Types:**
- `Config`: Parsed YAML configuration
- `Stock`: Individual asset with symbol, target percentage, and description
- `SymbolData`: Runtime data tracking current holdings, drift from target, and rebalancing needs

**Key Functions:**
- `rebalance()` (main.go:76): Reads CSV, calculates drift from target allocation, displays recommendations
- `deposit()` (main.go:195): Calculates how to split a deposit across assets
- `parseConfig()` (main.go:220): Loads YAML and validates percentages sum to 100

**Utilities:**
- `amountToInt()` (main.go:244): Parses dollar strings to cents (integer math avoids float precision issues)
- `formatAmount()` (main.go:254): Formats cents back to dollar strings with optional commas
- `green()/red()` (main.go:187,191): ANSI color codes for terminal output

### Amount Handling

All amounts are stored as **cents** (integers) to avoid floating-point precision errors. Dollar amounts are converted to cents immediately after parsing and converted back to dollar strings only for display.

### CSV Parsing

The tool expects CSV columns `Symbol` and `Current Value`:
- Fidelity CSVs work out-of-the-box
- Malformed lines at end of Fidelity CSVs are handled
- Only symbols listed in config are processed; others are ignored

### Drift Calculation

The core rebalancing logic (main.go:145-158):
1. Calculate current percentage: `(currentAmount / totalPortfolio) * 100`
2. Calculate drift: `currentPercentage - targetPercentage`
3. Calculate amount needed: `total * (-drift / 100)`

Positive drift = overweight (sell/hold), negative drift = underweight (buy).
