package controller

import (
	"context"
	logger "github.com/sirupsen/logrus"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
)

// OrderControllerKucoin executes the trading flow for KuCoin using the latest signal.
func OrderControllerKucoin(
	ctx context.Context,
	kucoinClient *connectors.KucoinConnector,
	user *model.User,
	orderSizePercent int,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string,
) error {

	logger.Debugf("OrderControllerKucoin INITIALIZED ")
	logger.Info("starting kucoin order controller flow")

	return nil
}
