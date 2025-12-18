package executors

import (
	"context"
	"errors"
	"fmt"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/controller"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"strategyexecutor/src/security"
	"time"

	logger "github.com/sirupsen/logrus"
)

var (
	newPhemexClient             = connectors.NewClient
	newGooeyClient              = connectors.NewGooeyClient
	newKrakenFuturesClient      = connectors.NewKrakenFuturesClient
	newKucoinConnector          = connectors.NewKucoinConnector
	orderControllerPhemex       = controller.OrderController
	orderControllerHydra        = controller.OrderControllerHydra
	orderControllerKrakenFuture = controller.OrderControllerKrakenFutures
	orderControllerKucoin       = controller.OrderControllerKucoin
)

func StartLoop(ctx context.Context) error {
	config := GetConfig()

	ticker := time.NewTicker(config.LoopPeriod) // Set up a ticker that fires periodically
	defer ticker.Stop()

	userName := config.UserID

	if userName == "" {
		return errors.New("user_id not set")
	}

	targetExchange := config.TargetExchange

	exchangeRep := repository.NewExchangeRepository()
	userExchangeRep := repository.NewUserExchangeRepository()
	userRep := repository.NewUserRepository()
	tvRepo := repository.NewTradingViewRepository()

	user, err := userRep.GetUserByUserName(ctx, userName)
	if err != nil {
		logger.
			WithField("userName", userName).
			Error("Failed to GetUserByUserName")
		return err
	}

	exchange, err := exchangeRep.FindByName(ctx, targetExchange)
	if err != nil {
		logger.WithError(err).Error("Failed to FindByName")
		return err
	}

	logger.Info("Check if the user has the necessary skills to run the strategy.")
	userExchange, err := userExchangeRep.GetByUserAndExchange(ctx, user.ID, exchange.ID)
	if err != nil || userExchange == nil {
		logger.WithError(err).Error("Failed to GetByUserAndExchange")
		return err
	}

	if userExchange.APIKeyHash == "" || userExchange.APISecretHash == "" {
		logger.Error("No valid key/secret set for exchange")
		return err
	}

	var apiPassphrase string
	if userExchange.APIPassphraseHash != "" {
		apiPassphrase, err = security.DecryptString(userExchange.APIPassphraseHash)
		if err != nil {
			logger.WithError(err).Error("Failed to decrypt API Passphrase")
			return err
		}
	}

	if config.TargetExchange == "kucoin" && apiPassphrase == "" {
		return errors.New("api passphrase not set for kucoin")
	}

	apiKey, err := security.DecryptString(userExchange.APIKeyHash)
	if err != nil {
		logger.WithError(err).Error("Failed to decrypt API Key")
		return err
	}
	apiSecret, err := security.DecryptString(userExchange.APISecretHash)
	if err != nil {
		logger.WithError(err).Error("Failed to decrypt API Secret")
		return err
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("loop stopped")
			return nil

		case <-ticker.C:
			logger.Info("loop tick")
			run, percent, err := getUserRunOnServerAndPercent(ctx, err, userExchangeRep, user.ID, exchange)
			if err != nil || !run {
				return errors.New("failed to get user run on server and percent or user run on server is false")
			}

			// A reasonable window: yesterday → tomorrow
			from := time.Now().Add(-12 * time.Hour).UTC()
			to := time.Now().Add(12 * time.Hour).UTC()

			tvLoaded, err := tvRepo.LoadImportantEventsFromDB(ctx, from, to, []string{"US"})
			if err != nil {
				return errors.New("failed to LoadImportantEventsFromDB")
			}

			cfg := connectors.NewNewsWindowConfig(15*time.Minute, 15*time.Minute)
			canEnterTrade := connectors.CanEnterTradeAt(time.Now(), tvLoaded, cfg)
			if !canEnterTrade.Allowed {
				return errors.New("trade window is not allowed")
			}

			err = runController(ctx, apiKey, apiSecret, apiPassphrase, user, percent, exchange)
			if err != nil {
				logger.WithError(err).Error("OrderController failed, will exit here")
				return err
			}

		}
	}
}

func getUserRunOnServerAndPercent(ctx context.Context, err error, userExchangeRep *repository.GormUserExchangeRepository, userID uint, exchange *model.Exchange) (bool, int, error) {
	run, percent, err := userExchangeRep.GetUserRunOnServerAndPercent(ctx, userID, exchange.ID)
	if err != nil {
		logger.WithError(err).
			WithField("user", userID).
			WithField("exchange", exchange.Name).
			Error("Failed to get GetUserRunOnServerAndPercent")
		return run, percent, err
	}
	return run, percent, nil
}

func runController(ctx context.Context, apiKey, apiSecret, apiPassphrase string, user *model.User, percent int, exchange *model.Exchange) error {
	config := GetConfig()
	baseURL := config.BaseURL
	targetExchange := config.TargetExchange
	targetSymbol := config.TargetSymbol

	// TODO: this should be an interface and the exchange specific implementation should be injected

	if targetExchange == "phemex" {
		phemexClient := newPhemexClient(apiKey, apiSecret, baseURL)
		err := orderControllerPhemex(ctx, phemexClient, user, percent, exchange.ID, targetSymbol, targetExchange)
		if err != nil {
			logger.WithError(err).Error("OrderController returned an error")
			return err
		}
	} else if targetExchange == "hydra" {
		c, err := newGooeyClient(apiKey, apiSecret)
		if err != nil {
			logger.WithError(err).Error("OrderController failed to start NewGooeyClient")
			return err
		}
		err = orderControllerHydra(ctx, c, user, exchange.ID, targetSymbol, targetExchange)
		if err != nil {
			logger.WithError(err).Error("OrderControllerHydra returned an error")
			return err
		}

	} else if targetExchange == "kraken" {
		c := newKrakenFuturesClient(apiKey, apiSecret, "")

		err := orderControllerKrakenFuture(ctx, c, user, exchange.ID, targetSymbol, targetExchange)
		if err != nil {
			logger.WithError(err).Error("OrderControllerKrakenFutures returned an error")
			return err
		}
	} else if targetExchange == "kucoin" {
		c := newKucoinConnector(apiKey, apiSecret, apiPassphrase, "")

		err := orderControllerKucoin(ctx, c, user, percent, exchange.ID, targetSymbol, targetExchange)
		if err != nil {
			logger.WithError(err).Error("OrderControllerKucoin returned an error")
			return err
		}
	} else {
		err := errors.New(fmt.Sprintf("exchange %s not supported", targetExchange))
		logger.WithError(err).Error("exchange not supported")
		return err
	}
	return nil
}
