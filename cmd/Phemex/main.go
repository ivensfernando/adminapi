package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strategyexecutor/src/connectors"
	"strconv"
	"strings"

	logger "github.com/sirupsen/logrus"
)

func SetupLogger() {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))

	level, err := logger.ParseLevel(levelStr)
	if err != nil {
		level = logger.DebugLevel // fallback seguro
	}

	logger.SetLevel(level)
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	logger.WithField("level", level.String()).
		Info("Logger initialized for Phemex CLI")
}

func printUsage() {
	fmt.Println("Available commands:")
	fmt.Println("  help                             Show this help message")
	fmt.Println("  shutdown                         Exit the application")
	fmt.Println("  positions                        List all USDT-M positions")
	fmt.Println("  long SYMBOL QTY                  Open LONG market position")
	fmt.Println("  short SYMBOL QTY                 Open SHORT market position")
	fmt.Println("  close-long SYMBOL QTY            Close LONG")
	fmt.Println("  close-short SYMBOL QTY           Close SHORT")
	fmt.Println("  reverse SYMBOL QTY               Reverse position")
	fmt.Println("  cancel-all SYMBOL                Cancel all orders")
	fmt.Println("  cancel-all-positions SYMBOL      Cancel all positions for a symbol (including open orders)")
	fmt.Println("  ticker SYMBOL                    Show ticker info")
	fmt.Println("  orderbook SYMBOL                 Show orderbook")
	fmt.Println("  orders SYMBOL                    Show active orders")
	fmt.Println("  ordershistory SYMBOL             Show order history")
	fmt.Println("  fills SYMBOL                     Show fills")
	fmt.Println("  klines SYMBOL RESOLUTION         Show klines")
	fmt.Println("  disp SYMBOL                      Show available USDT margin for symbol")
	fmt.Println("  avl SYMBOL                       Show available base coin from USDT margin")
	fmt.Println()
}

func printJSON(data any) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("failed to marshal JSON for printing")
		fmt.Println("JSON error:", err)
		return
	}
	fmt.Println(string(b))
}

func printPositions(pos *connectors.GAccountPositions) {
	fmt.Printf("USDT Balance: %s\n", pos.Account.AccountBalanceRv)

	found := false

	for _, p := range pos.Positions {
		if p.SizeRq == "" || p.SizeRq == "0" {
			continue
		}

		found = true
		fmt.Println("------ OPEN POSITION ------")
		fmt.Printf("Symbol:     %s\n", p.Symbol)
		fmt.Printf("PosSide:    %s\n", p.PosSide)
		fmt.Printf("SizeRq:     %s\n", p.SizeRq)
		fmt.Printf("AvgPrice:   %s\n", p.AvgEntryPriceRp)
		fmt.Printf("Margin:     %s\n", p.PositionMarginRv)
		fmt.Printf("MarkPrice:  %s\n", p.MarkPriceRp)
		fmt.Println("---------------------------")
	}

	if !found {
		fmt.Println("No open USDT-M positions.")
	}
}

func printOrders(data json.RawMessage) {
	var payload struct {
		Rows    []map[string]interface{} `json:"rows"`
		HasNext bool                     `json:"hasNext"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		logger.WithError(err).Error("failed to parse orders payload")
		fmt.Println("Error parsing orders:", err)
		printJSON(data)
		return
	}

	if len(payload.Rows) == 0 {
		fmt.Println("No active orders.")
		return
	}

	for i, row := range payload.Rows {
		fmt.Printf("------ ORDER %d ------\n", i+1)
		printMapField(row, "symbol", "Symbol")
		printMapField(row, "side", "Side")
		printMapField(row, "posSide", "PosSide")
		printMapField(row, "ordType", "OrdType")
		printMapField(row, "priceRp", "PriceRp")
		printMapField(row, "orderQtyRq", "QtyRq")
		printMapField(row, "reduceOnly", "ReduceOnly")
		printMapField(row, "ordStatus", "Status")
		printMapField(row, "clOrdID", "ClientID")
		printMapField(row, "cumQtyRq", "FilledQty")
		printMapField(row, "leavesQtyRq", "LeavesQty")
		printMapField(row, "stopPxRp", "StopPx")
		fmt.Println("---------------------")
	}

	if payload.HasNext {
		fmt.Println("More orders available...")
	}
}

func printOrderbook(data json.RawMessage) {
	var payload struct {
		Depth      int    `json:"depth"`
		Dts        int64  `json:"dts"`
		Mts        int64  `json:"mts"`
		Timestamp  int64  `json:"timestamp"`
		Sequence   int64  `json:"sequence"`
		Symbol     string `json:"symbol"`
		Type       string `json:"type"`
		OrderbookP struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
		} `json:"orderbook_p"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		logger.WithError(err).Error("failed to parse orderbook payload")
		fmt.Println("Error parsing orderbook:", err)
		printJSON(data)
		return
	}

	fmt.Println("------ ORDERBOOK ------")
	fmt.Printf("Symbol: %s\n", payload.Symbol)
	fmt.Printf("Timestamp: %d\n", payload.Timestamp)

	fmt.Println("Asks:")
	for _, lvl := range payload.OrderbookP.Asks {
		if len(lvl) < 2 {
			continue
		}
		fmt.Printf("  Price: %s  Qty: %s\n", lvl[0], lvl[1])
	}

	fmt.Println("Bids:")
	for _, lvl := range payload.OrderbookP.Bids {
		if len(lvl) < 2 {
			continue
		}
		fmt.Printf("  Price: %s  Qty: %s\n", lvl[0], lvl[1])
	}

	fmt.Println("-----------------------")
}

func printLevels(label string, levels [][]json.RawMessage) {
	fmt.Printf("%s (top 5):\n", label)
	if len(levels) == 0 {
		fmt.Println("  none")
		return
	}

	limit := 5
	if len(levels) < limit {
		limit = len(levels)
	}

	for i := 0; i < limit; i++ {
		var parts []string
		for _, raw := range levels[i] {
			parts = append(parts, strings.Trim(string(raw), "\""))
		}
		fmt.Printf("  %d) %s\n", i+1, strings.Join(parts, " | "))
	}
}

func printMapField(m map[string]interface{}, key, label string) {
	if v, ok := m[key]; ok {
		fmt.Printf("%-11s: %v\n", label, v)
	}
}

func main() {
	SetupLogger()

	apiKey := os.Getenv("PHEMEX_API_KEY")
	apiSecret := os.Getenv("PHEMEX_API_SECRET")
	baseURL := os.Getenv("PHEMEX_BASE_URL")

	if baseURL == "" {
		baseURL = "https://testnet-api.phemex.com"
		logger.Warnf("No base URL provided, using default: %s", baseURL)
	}

	if apiKey == "" || apiSecret == "" {
		logger.Fatal("Missing API keys (PHEMEX_API_KEY / PHEMEX_API_SECRET)")
	}

	client := connectors.NewClient(apiKey, apiSecret, baseURL)

	reader := bufio.NewScanner(os.Stdin)
	fmt.Println("Phemex CLI Ready. Type 'help' for a list of commands. Type 'shutdown' to exit.")
	logger.Info("Phemex CLI started")

	for {
		fmt.Print("phemex> ")

		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				logger.WithError(err).Error("stdin scanner error")
			}
			continue
		}

		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]

		logger.WithField("command_line", line).Debug("Received CLI command")

		switch cmd {

		case "shutdown":
			logger.Info("Shutdown command received, exiting CLI")
			fmt.Println("Exiting CLI...")
			return

		case "help":
			printUsage()

		case "positions":
			logger.Info("Listing USDT-M positions")
			pos, err := client.GetPositionsUSDT()
			if err != nil {
				logger.WithError(err).Error("failed to get positions")
				fmt.Println("Error:", err)
				continue
			}
			printPositions(pos)

		case "long":
			if len(parts) < 3 {
				fmt.Println("Usage: long SYMBOL QTY")
				printUsage()
				continue
			}
			symbol, qty := parts[1], parts[2]

			logger.WithFields(logger.Fields{
				"cmd":    "long",
				"symbol": symbol,
				"qty":    qty,
			}).Info("Executing LONG order")

			fmt.Printf("Executing LONG %s qty=%s\n", symbol, qty)

			resp, err := client.PlaceOrder(symbol, "Buy", "Long", qty, "Market", false)
			if err != nil {
				logger.WithError(err).Error("failed to place LONG order")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "short":
			if len(parts) < 3 {
				fmt.Println("Usage: short SYMBOL QTY")
				printUsage()
				continue
			}
			symbol, qty := parts[1], parts[2]

			logger.WithFields(logger.Fields{
				"cmd":    "short",
				"symbol": symbol,
				"qty":    qty,
			}).Info("Executing SHORT order")

			fmt.Printf("Executing SHORT %s qty=%s\n", symbol, qty)

			resp, err := client.PlaceOrder(symbol, "Sell", "Short", qty, "Market", false)
			if err != nil {
				logger.WithError(err).Error("failed to place SHORT order")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "close-long":
			if len(parts) < 3 {
				fmt.Println("Usage: close-long SYMBOL QTY")
				printUsage()
				continue
			}
			symbol, qty := parts[1], parts[2]

			logger.WithFields(logger.Fields{
				"cmd":    "close-long",
				"symbol": symbol,
				"qty":    qty,
			}).Info("Closing LONG position")

			fmt.Printf("Closing LONG %s qty=%s\n", symbol, qty)

			resp, err := client.PlaceOrder(symbol, "Sell", "Long", qty, "Market", true)
			if err != nil {
				logger.WithError(err).Error("failed to close LONG position")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "close-short":
			if len(parts) < 3 {
				fmt.Println("Usage: close-short SYMBOL QTY")
				printUsage()
				continue
			}
			symbol, qty := parts[1], parts[2]

			logger.WithFields(logger.Fields{
				"cmd":    "close-short",
				"symbol": symbol,
				"qty":    qty,
			}).Info("Closing SHORT position")

			fmt.Printf("Closing SHORT %s qty=%s\n", symbol, qty)

			resp, err := client.PlaceOrder(symbol, "Buy", "Short", qty, "Market", true)
			if err != nil {
				logger.WithError(err).Error("failed to close SHORT position")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "reverse":
			if len(parts) < 3 {
				fmt.Println("Usage: reverse SYMBOL QTY")
				printUsage()
				continue
			}
			symbol, qty := parts[1], parts[2]

			logger.WithFields(logger.Fields{
				"cmd":    "reverse",
				"symbol": symbol,
				"qty":    qty,
			}).Info("Reversing position")

			fmt.Printf("Reversing %s qty=%s\n", symbol, qty)

			// Close LONG side
			if _, err := client.PlaceOrder(symbol, "Sell", "Long", qty, "Market", true); err != nil {
				logger.WithError(err).Error("failed to close LONG part of reverse")
				fmt.Println("Error closing LONG:", err)
				continue
			}

			// Open SHORT side
			resp, err := client.PlaceOrder(symbol, "Sell", "Short", qty, "Market", false)
			if err != nil {
				logger.WithError(err).Error("failed to open SHORT part of reverse")
				fmt.Println("Error opening SHORT:", err)
				continue
			}
			printJSON(resp.Data)

		case "cancel-all":
			if len(parts) < 2 {
				fmt.Println("Usage: cancel-all SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "cancel-all",
				"symbol": symbol,
			}).Info("Canceling all orders for symbol")

			resp, err := client.CancelAll(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to cancel all orders")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "cancel-all-positions":
			if len(parts) < 2 {
				fmt.Println("Usage: cancel-all-positions SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "cancel-all-positions",
				"symbol": symbol,
			}).Info("Closing all positions for symbol")

			err := client.CloseAllPositions(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to close all positions")
				fmt.Println("Error:", err)
				continue
			}

			pos, err := client.GetPositionsUSDT()
			if err != nil {
				logger.WithError(err).Error("failed to fetch positions after closing")
				fmt.Println("Error:", err)
				continue
			}
			printPositions(pos)

		case "ticker":
			if len(parts) < 2 {
				fmt.Println("Usage: ticker SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "ticker",
				"symbol": symbol,
			}).Info("Fetching ticker")

			resp, err := client.GetTicker(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch ticker")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "orderbook":
			if len(parts) < 2 {
				fmt.Println("Usage: orderbook SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "orderbook",
				"symbol": symbol,
			}).Info("Fetching orderbook")

			resp, err := client.GetOrderbook(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch orderbook")
				fmt.Println("Error:", err)
				continue
			}
			printOrderbook(resp.Data)

		case "orders":
			if len(parts) < 2 {
				fmt.Println("Usage: orders SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "orders",
				"symbol": symbol,
			}).Info("Fetching active orders")

			resp, err := client.GetActiveOrders(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch active orders")
				fmt.Println("Error:", err)
				continue
			}
			printOrders(resp.Data)

		case "ordershistory":
			if len(parts) < 2 {
				fmt.Println("Usage: ordershistory SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "ordershistory",
				"symbol": symbol,
			}).Info("Fetching order history")

			resp, err := client.GetOrderHistory(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch order history")
				fmt.Println("Error:", err)
				continue
			}
			printOrders(resp.Data)

		case "fills":
			if len(parts) < 2 {
				fmt.Println("Usage: fills SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "fills",
				"symbol": symbol,
			}).Info("Fetching fills")

			resp, err := client.GetFills(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch fills")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "klines":
			if len(parts) < 3 {
				fmt.Println("Usage: klines SYMBOL RESOLUTION")
				printUsage()
				continue
			}
			symbol := parts[1]
			res, _ := strconv.Atoi(parts[2])

			logger.WithFields(logger.Fields{
				"cmd":        "klines",
				"symbol":     symbol,
				"resolution": res,
			}).Info("Fetching klines")

			resp, err := client.GetKlines(symbol, res)
			if err != nil {
				logger.WithError(err).Error("failed to fetch klines")
				fmt.Println("Error:", err)
				continue
			}
			printJSON(resp.Data)

		case "disp":
			if len(parts) < 2 {
				fmt.Println("Usage: disp SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "disp",
				"symbol": symbol,
			}).Info("Fetching available USDT margin from risk-unit")

			qtd, err := client.GetFuturesAvailableFromRiskUnit(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to fetch available USDT margin")
				fmt.Println("Error:", err)
				continue
			}

			fmt.Printf("USDT available %.12f\n", qtd)

		case "avl":
			if len(parts) < 2 {
				fmt.Println("Usage: avl SYMBOL")
				printUsage()
				continue
			}
			symbol := parts[1]

			logger.WithFields(logger.Fields{
				"cmd":    "avl",
				"symbol": symbol,
			}).Info("Fetching base availability from USDT margin")

			baseSymbol, baseAvail, usdtAvail, price, err := client.GetAvailableBaseFromUSDT(symbol)
			if err != nil {
				logger.WithError(err).Error("failed to compute base availability from USDT margin")
				fmt.Println("Error:", err)
				continue
			}

			fmt.Printf("Available %s\n", baseSymbol)
			fmt.Printf("USDT -> base coin %.12f\n", baseAvail)
			fmt.Printf("USDT available %.12f\n", usdtAvail)
			fmt.Printf("USDT price %.12f\n", price)

		default:
			logger.WithField("cmd", cmd).Warn("Unknown command received")
			fmt.Println("Unknown command:", cmd)
			printUsage()
		}
	}
}
