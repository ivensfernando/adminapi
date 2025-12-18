package executors

import (
	"context"
	"errors"
	"testing"

	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
)

// Ensures KuCoin branch uses the connector and controller with decrypted passphrase.
func TestRunControllerKucoin(t *testing.T) {
	oldNewConnector := newKucoinConnector
	oldController := orderControllerKucoin
	t.Cleanup(func() {
		newKucoinConnector = oldNewConnector
		orderControllerKucoin = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "kucoin")
	t.Setenv("TARGET_SYMBOL", "XBTUSDTM")

	calledConnector := false
	newKucoinConnector = func(apiKey, apiSecret, apiPassphrase, keyVersion string) *connectors.KucoinConnector {
		calledConnector = true
		if apiKey != "key" || apiSecret != "secret" || apiPassphrase != "pass" || keyVersion != "" {
			t.Fatalf("unexpected credentials passed to kucoin connector: %s %s %s %s", apiKey, apiSecret, apiPassphrase, keyVersion)
		}
		return &connectors.KucoinConnector{}
	}

	calledController := false
	orderControllerKucoin = func(ctx context.Context, kucoinClient *connectors.KucoinConnector, user *model.User, orderSizePercent int, exchangeID uint, targetSymbol string, targetExchange string) error {
		calledController = true
		if orderSizePercent != 25 || exchangeID != 42 || targetSymbol != "XBTUSDTM" || targetExchange != "kucoin" {
			t.Fatalf("unexpected parameters passed to kucoin controller: %d %d %s %s", orderSizePercent, exchangeID, targetSymbol, targetExchange)
		}
		return nil
	}

	err := runController(context.Background(), "key", "secret", "pass", &model.User{ID: 1}, 25, &model.Exchange{ID: 42, Name: "kucoin"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !calledConnector {
		t.Fatalf("expected kucoin connector to be created")
	}

	if !calledController {
		t.Fatalf("expected kucoin controller to be invoked")
	}
}

// Ensures errors from the KuCoin controller propagate back to the caller.
func TestRunControllerKucoinError(t *testing.T) {
	oldNewConnector := newKucoinConnector
	oldController := orderControllerKucoin
	t.Cleanup(func() {
		newKucoinConnector = oldNewConnector
		orderControllerKucoin = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "kucoin")
	t.Setenv("TARGET_SYMBOL", "ETHUSDTM")

	newKucoinConnector = func(apiKey, apiSecret, apiPassphrase, keyVersion string) *connectors.KucoinConnector {
		return &connectors.KucoinConnector{}
	}

	orderControllerKucoin = func(ctx context.Context, kucoinClient *connectors.KucoinConnector, user *model.User, orderSizePercent int, exchangeID uint, targetSymbol string, targetExchange string) error {
		return errors.New("kucoin controller failed")
	}

	err := runController(context.Background(), "key", "secret", "pass", &model.User{ID: 2}, 10, &model.Exchange{ID: 99, Name: "kucoin"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Verifies the Phemex branch builds a client with the provided credentials and routes to the controller.
func TestRunControllerPhemex(t *testing.T) {
	oldNewClient := newPhemexClient
	oldController := orderControllerPhemex
	t.Cleanup(func() {
		newPhemexClient = oldNewClient
		orderControllerPhemex = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "phemex")
	t.Setenv("TARGET_SYMBOL", "BTCUSDT")

	clientBuilt := false
	newPhemexClient = func(apiKey, apiSecret, baseURL string) *connectors.Client {
		clientBuilt = true
		if apiKey != "pkey" || apiSecret != "psecret" || baseURL != "https://example" {
			t.Fatalf("unexpected Phemex credentials: %s %s %s", apiKey, apiSecret, baseURL)
		}
		return &connectors.Client{}
	}

	controllerCalled := false
	orderControllerPhemex = func(ctx context.Context, phemexClient *connectors.Client, user *model.User, orderSizePercent int, exchangeID uint, targetSymbol string, targetExchange string) error {
		controllerCalled = true
		if orderSizePercent != 15 || exchangeID != 7 || targetSymbol != "BTCUSDT" || targetExchange != "phemex" {
			t.Fatalf("unexpected Phemex controller params: %d %d %s %s", orderSizePercent, exchangeID, targetSymbol, targetExchange)
		}
		return nil
	}

	if err := runController(context.Background(), "pkey", "psecret", "", &model.User{ID: 10}, 15, &model.Exchange{ID: 7, Name: "phemex"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !clientBuilt {
		t.Fatalf("expected Phemex client to be built")
	}

	if !controllerCalled {
		t.Fatalf("expected Phemex controller to be called")
	}
}

// Ensures Phemex controller errors are surfaced to the caller.
func TestRunControllerPhemexError(t *testing.T) {
	oldNewClient := newPhemexClient
	oldController := orderControllerPhemex
	t.Cleanup(func() {
		newPhemexClient = oldNewClient
		orderControllerPhemex = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "phemex")
	t.Setenv("TARGET_SYMBOL", "BTCUSD")

	newPhemexClient = func(apiKey, apiSecret, baseURL string) *connectors.Client {
		return &connectors.Client{}
	}

	orderControllerPhemex = func(ctx context.Context, phemexClient *connectors.Client, user *model.User, orderSizePercent int, exchangeID uint, targetSymbol string, targetExchange string) error {
		return errors.New("phemex failure")
	}

	if err := runController(context.Background(), "k", "s", "", &model.User{ID: 2}, 50, &model.Exchange{ID: 3, Name: "phemex"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Verifies the Hydra branch starts the Gooey client and routes to the controller.
func TestRunControllerHydra(t *testing.T) {
	oldNewClient := newGooeyClient
	oldController := orderControllerHydra
	t.Cleanup(func() {
		newGooeyClient = oldNewClient
		orderControllerHydra = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "hydra")
	t.Setenv("TARGET_SYMBOL", "SOLUSD")

	clientBuilt := false
	newGooeyClient = func(apiKey, apiSecret string) (*connectors.GooeyClient, error) {
		clientBuilt = true
		if apiKey != "gkey" || apiSecret != "gsecret" {
			t.Fatalf("unexpected Hydra credentials: %s %s", apiKey, apiSecret)
		}
		return &connectors.GooeyClient{}, nil
	}

	controllerCalled := false
	orderControllerHydra = func(ctx context.Context, c *connectors.GooeyClient, user *model.User, exchangeID uint, targetSymbol string, targetExchange string) error {
		controllerCalled = true
		if exchangeID != 11 || targetSymbol != "SOLUSD" || targetExchange != "hydra" {
			t.Fatalf("unexpected Hydra controller params: %d %s %s", exchangeID, targetSymbol, targetExchange)
		}
		return nil
	}

	if err := runController(context.Background(), "gkey", "gsecret", "", &model.User{ID: 4}, 0, &model.Exchange{ID: 11, Name: "hydra"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !clientBuilt {
		t.Fatalf("expected Hydra client to be built")
	}

	if !controllerCalled {
		t.Fatalf("expected Hydra controller to be called")
	}
}

// Ensures Hydra controller errors propagate after client creation succeeds.
func TestRunControllerHydraError(t *testing.T) {
	oldNewClient := newGooeyClient
	oldController := orderControllerHydra
	t.Cleanup(func() {
		newGooeyClient = oldNewClient
		orderControllerHydra = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "hydra")
	t.Setenv("TARGET_SYMBOL", "DOGEUSD")

	newGooeyClient = func(apiKey, apiSecret string) (*connectors.GooeyClient, error) {
		return &connectors.GooeyClient{}, nil
	}

	orderControllerHydra = func(ctx context.Context, c *connectors.GooeyClient, user *model.User, exchangeID uint, targetSymbol string, targetExchange string) error {
		return errors.New("hydra failed")
	}

	if err := runController(context.Background(), "g", "s", "", &model.User{ID: 8}, 0, &model.Exchange{ID: 9, Name: "hydra"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Verifies the Kraken branch builds the futures client and routes to the controller.
func TestRunControllerKraken(t *testing.T) {
	oldNewClient := newKrakenFuturesClient
	oldController := orderControllerKrakenFuture
	t.Cleanup(func() {
		newKrakenFuturesClient = oldNewClient
		orderControllerKrakenFuture = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "kraken")
	t.Setenv("TARGET_SYMBOL", "ETHUSD")

	clientBuilt := false
	newKrakenFuturesClient = func(apiKey, apiSecret, baseURL string) *connectors.KrakenFuturesClient {
		clientBuilt = true
		if apiKey != "kkey" || apiSecret != "ksecret" || baseURL != "" {
			t.Fatalf("unexpected Kraken credentials: %s %s %s", apiKey, apiSecret, baseURL)
		}
		return &connectors.KrakenFuturesClient{}
	}

	controllerCalled := false
	orderControllerKrakenFuture = func(ctx context.Context, c *connectors.KrakenFuturesClient, user *model.User, exchangeID uint, targetSymbol string, targetExchange string) error {
		controllerCalled = true
		if exchangeID != 12 || targetSymbol != "ETHUSD" || targetExchange != "kraken" {
			t.Fatalf("unexpected Kraken controller params: %d %s %s", exchangeID, targetSymbol, targetExchange)
		}
		return nil
	}

	if err := runController(context.Background(), "kkey", "ksecret", "", &model.User{ID: 6}, 0, &model.Exchange{ID: 12, Name: "kraken"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !clientBuilt {
		t.Fatalf("expected Kraken client to be built")
	}

	if !controllerCalled {
		t.Fatalf("expected Kraken controller to be called")
	}
}

// Ensures Kraken controller errors bubble up to the caller.
func TestRunControllerKrakenError(t *testing.T) {
	oldNewClient := newKrakenFuturesClient
	oldController := orderControllerKrakenFuture
	t.Cleanup(func() {
		newKrakenFuturesClient = oldNewClient
		orderControllerKrakenFuture = oldController
	})

	t.Setenv("TARGET_EXCHANGE", "kraken")
	t.Setenv("TARGET_SYMBOL", "ETHUSD")

	newKrakenFuturesClient = func(apiKey, apiSecret, baseURL string) *connectors.KrakenFuturesClient {
		return &connectors.KrakenFuturesClient{}
	}

	orderControllerKrakenFuture = func(ctx context.Context, c *connectors.KrakenFuturesClient, user *model.User, exchangeID uint, targetSymbol string, targetExchange string) error {
		return errors.New("kraken failed")
	}

	if err := runController(context.Background(), "k", "s", "", &model.User{ID: 6}, 0, &model.Exchange{ID: 5, Name: "kraken"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Ensures unsupported exchanges return an explicit error before any connector creation.
func TestRunControllerUnsupportedExchange(t *testing.T) {
	t.Setenv("TARGET_EXCHANGE", "unknown")

	if err := runController(context.Background(), "k", "s", "", &model.User{ID: 1}, 0, &model.Exchange{ID: 1, Name: "unknown"}); err == nil {
		t.Fatalf("expected error for unsupported exchange")
	}
}
