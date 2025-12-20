package executors

import (
	"context"
	"errors"
	"fmt"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/controller"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"strategyexecutor/src/risk"
	"strategyexecutor/src/security"
	"time"

	"github.com/shopspring/decimal"
	logger "github.com/sirupsen/logrus"
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

	logger.Info("GetByUserAndExchange call. get user exchange setting, check strategy enabled, verify key/secret")
	userExchange, err := userExchangeRep.GetByUserAndExchange(ctx, user.ID, exchange.ID)
	if err != nil || userExchange == nil {
		logger.WithError(err).Error("Failed to GetByUserAndExchange")
		return err
	}

	if userExchange.APIKeyHash == "" || userExchange.APISecretHash == "" {
		logger.Error("No valid key/secret set for exchange")
		return err
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
			logger.Info("GetByUserAndExchange call. get user exchange setting, check strategy enabled, verify key/secret")
			userExchange, err = userExchangeRep.GetByUserAndExchange(ctx, user.ID, exchange.ID)
			if err != nil || userExchange == nil {
				logger.WithError(err).Error("Failed to GetByUserAndExchange")
				return err
			}
			run := userExchange.RunOnServer
			if !run {
				logger.Warn("strategy disabled, skipping")
				return nil
			}

			// check risk off mode
			cfg := risk.NewSessionSizeConfigFromUserExchangeOrDefault(userExchange)
			_, session := risk.CalculateSizeByNYSession(
				decimal.Zero,
				time.Now(),
				cfg,
			)

			if session == risk.SessionNoTrade {
				logger.Warn(risk.SessionNoTrade + " - risk off mode")

				if userExchange.NoTradeWindowOrdersClosed {
					logger.Warn("no trade window orders already closed, short circuiting")
					return nil
				} else {
					logger.Warn("no trade window orders not yet closed, will continue with the loop")
				}
			}

			// check if news window -> risk off mode

			// fetch news for a reasonable window: yesterday → tomorrow
			from := time.Now().Add(-12 * time.Hour).UTC()
			to := time.Now().Add(12 * time.Hour).UTC()

			tvLoaded, err := tvRepo.LoadImportantEventsFromDB(ctx, from, to, []string{"US"})
			if err != nil {
				return errors.New("failed to LoadImportantEventsFromDB")
			}

			newsCfg := connectors.NewNewsWindowConfig(15*time.Minute, 15*time.Minute)
			canEnterTrade := connectors.CanEnterTradeAt(time.Now(), tvLoaded, newsCfg)
			if !canEnterTrade.Allowed {
				return errors.New("trade window is not allowed")
			}

			err = runController(ctx, apiKey, apiSecret, user, userExchange, exchange)
			if err != nil {
				logger.WithError(err).Error("OrderController failed, will exit here")
				return err
			}

		}
	}
}

func runController(ctx context.Context, apiKey, apiSecret string, user *model.User, userExchange *model.UserExchange, exchange *model.Exchange) error {
	config := GetConfig()
	baseURL := config.BaseURL
	targetExchange := config.TargetExchange
	targetSymbol := config.TargetSymbol

	// TODO: this should be an interface and the exchange specific implementation should be injected

	if targetExchange == "phemex" {
		phemexClient := connectors.NewClient(apiKey, apiSecret, baseURL)
		err := controller.OrderController(ctx, phemexClient, user, exchange.ID, targetSymbol, targetExchange, userExchange)
		if err != nil {
			logger.WithError(err).Error("OrderController returned an error")
			return err
		}
	} else if targetExchange == "hydra" {
		c, err := connectors.NewGooeyClient(apiKey, apiSecret)
		if err != nil {
			logger.WithError(err).Error("OrderController failed to start NewGooeyClient")
			return err
		}
		err = controller.OrderControllerHydra(ctx, c, user, exchange.ID, targetSymbol, targetExchange, userExchange)
		if err != nil {
			logger.WithError(err).Error("OrderControllerHydra returned an error")
			return err
		}

	} else if targetExchange == "kraken" {
		c := connectors.NewKrakenFuturesClient(apiKey, apiSecret, "")
		err := controller.OrderControllerKrakenFutures(ctx, c, user, exchange.ID, targetSymbol, targetExchange, userExchange)
		if err != nil {
			logger.WithError(err).Error("OrderControllerKrakenFutures returned an error")
			return err
		}
	} else {
		err := errors.New(fmt.Sprintf("exchange %s not supported", targetExchange))
		logger.WithError(err).Error("exchange not supported")
		return err
	}
	return nil
}
