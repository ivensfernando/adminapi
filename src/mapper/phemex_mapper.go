package mapper

import (
	"strconv"
	"time"

	logger "github.com/sirupsen/logrus"

	"strategyexecutor/src/model"
)

// MapPhemexResponseToModel converts a raw Phemex API response into a database model
// in a "safe" way: parsing errors on numeric fields are logged and defaulted to 0,
// instead of aborting the whole mapping.
func MapPhemexResponseToModel(
	resp *model.PhemexOrderResponse,
	internalOrderID uint,
) (*model.PhemexOrder, error) {

	if resp == nil {
		logger.WithField("mapper", "MapPhemexResponseToModel").
			Error("Nil PhemexOrderResponse received")
		return nil, nil
	}

	logger.WithFields(map[string]interface{}{
		"mapper":           "MapPhemexResponseToModel",
		"internal_orderID": internalOrderID,
		"exchange_orderID": resp.OrderID,
		"symbol":           resp.Symbol,
		"side":             resp.Side,
	}).Debug("Safely mapping Phemex response to DB model")

	parseFloatSafe := func(field, v string) float64 {
		if v == "" {
			logger.WithFields(map[string]interface{}{
				"field": field,
			}).Debug("Empty numeric field received, defaulting to 0")
			return 0
		}

		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"field": field,
				"value": v,
			}).WithError(err).Error("Failed to parse float from Phemex response field; defaulting to 0")
			return 0
		}
		return f
	}

	price := parseFloatSafe("PriceRp", resp.PriceRp)
	orderQty := parseFloatSafe("OrderQtyRq", resp.OrderQtyRq)
	displayQty := parseFloatSafe("DisplayQtyRq", resp.DisplayQtyRq)

	closedPnl := parseFloatSafe("ClosedPnlRv", resp.ClosedPnlRv)
	closedSize := parseFloatSafe("ClosedSizeRq", resp.ClosedSizeRq)

	cumQty := parseFloatSafe("CumQtyRq", resp.CumQtyRq)
	cumValue := parseFloatSafe("CumValueRv", resp.CumValueRv)

	leavesQty := parseFloatSafe("LeavesQtyRq", resp.LeavesQtyRq)
	leavesValue := parseFloatSafe("LeavesValueRv", resp.LeavesValueRv)

	stopPrice := parseFloatSafe("StopPxRp", resp.StopPxRp)

	pegOffsetValue := parseFloatSafe("PegOffsetValueRp", resp.PegOffsetValueRp)
	pegOffsetProp := parseFloatSafe("PegOffsetProportionRr", resp.PegOffsetProportionRr)

	takeProfit := parseFloatSafe("TakeProfitRp", resp.TakeProfitRp)
	stopLoss := parseFloatSafe("StopLossRp", resp.StopLossRp)

	slPrice := parseFloatSafe("SlPxRp", resp.SlPxRp)
	tpPrice := parseFloatSafe("TpPxRp", resp.TpPxRp)

	// Convert nanoseconds to time.Time (se vier 0, vira epoch)
	actionTime := time.Unix(0, resp.ActionTimeNs)
	transactTime := time.Unix(0, resp.TransactTimeNs)

	order := &model.PhemexOrder{
		OrderID:         internalOrderID,
		ExchangeOrderID: resp.OrderID,
		ClOrdID:         resp.ClOrdID,
		Symbol:          resp.Symbol,
		Side:            resp.Side,

		ActionTime:   actionTime,
		TransactTime: transactTime,

		OrderType:   resp.OrderType,
		Price:       price,
		OrderQty:    orderQty,
		DisplayQty:  displayQty,
		TimeInForce: resp.TimeInForce,

		ClosedPnl:  closedPnl,
		ClosedSize: closedSize,

		CumQty:   cumQty,
		CumValue: cumValue,

		LeavesQty:   leavesQty,
		LeavesValue: leavesValue,

		StopDirection: resp.StopDirection,
		StopPrice:     stopPrice,
		Trigger:       resp.Trigger,

		PegOffsetValue:      pegOffsetValue,
		PegOffsetProportion: pegOffsetProp,
		PegPriceType:        resp.PegPriceType,

		ExecStatus: resp.ExecStatus,
		OrdStatus:  resp.OrdStatus,
		ExecInst:   resp.ExecInst,

		TakeProfit: takeProfit,
		StopLoss:   stopLoss,
		SlPrice:    slPrice,
		TpPrice:    tpPrice,
	}

	logger.WithFields(map[string]interface{}{
		"mapper":           "MapPhemexResponseToModel",
		"internal_orderID": internalOrderID,
		"exchange_orderID": resp.OrderID,
		"symbol":           resp.Symbol,
		"side":             resp.Side,
		"price":            price,
		"order_qty":        orderQty,
	}).Info("Phemex response safely mapped to model")

	return order, nil
}
