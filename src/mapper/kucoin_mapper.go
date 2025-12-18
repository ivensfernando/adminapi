package mapper

import (
	"strconv"
	"time"

	logger "github.com/sirupsen/logrus"

	"adminapi/src/model"
)

// MapKucoinResponseToModel converts a KuCoin futures order response into a normalized model.
func MapKucoinResponseToModel(resp *model.KucoinOrderResponse, internalOrderID uint) (*model.KucoinOrder, error) {
	if resp == nil {
		logger.WithField("mapper", "MapKucoinResponseToModel").
			Error("Nil KucoinOrderResponse received")
		return nil, nil
	}

	parseFloatSafe := func(field, v string) float64 {
		if v == "" {
			logger.WithField("field", field).Debug("Empty numeric field received, defaulting to 0")
			return 0
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"field": field,
				"value": v,
			}).WithError(err).Error("Failed to parse float from KuCoin response field; defaulting to 0")
			return 0
		}
		return f
	}

	orderTime := time.Unix(0, 0)
	if resp.OrderTime > 0 {
		// KuCoin returns milliseconds
		orderTime = time.UnixMilli(resp.OrderTime)
	}

	order := &model.KucoinOrder{
		OrderID:         internalOrderID,
		ExchangeOrderID: resp.OrderID,
		ClientOid:       resp.ClientOid,
		Symbol:          resp.Symbol,
		Side:            resp.Side,
		OrderType:       resp.Type,
		Status:          resp.Status,
		Price:           parseFloatSafe("price", resp.Price),
		Size:            parseFloatSafe("size", resp.Size),
		FilledSize:      parseFloatSafe("dealSize", resp.DealSize),
		FilledValue:     parseFloatSafe("dealValue", resp.DealValue),
		Leverage:        parseFloatSafe("leverage", resp.Leverage),
		Fee:             parseFloatSafe("fee", resp.Fee),
		FeeCurrency:     resp.FeeCurrency,
		TimeInForce:     resp.TimeInForce,
		Remark:          resp.Remark,
		OrderTime:       orderTime,
	}

	logger.WithFields(map[string]interface{}{
		"mapper":           "MapKucoinResponseToModel",
		"internal_orderID": internalOrderID,
		"exchange_orderID": resp.OrderID,
		"symbol":           resp.Symbol,
		"side":             resp.Side,
	}).Info("KuCoin response safely mapped to model")

	return order, nil
}
