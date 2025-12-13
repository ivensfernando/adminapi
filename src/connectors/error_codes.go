package connectors

import "fmt"

// PhemexErrorCodes maps Phemex bizError codes to human-readable messages.
var PhemexErrorCodes = map[int]string{
	11001: "TE_SUCCESS",                        // No error, success
	11002: "TE_UNKNOWN_ERROR",                  // Unknown error
	11003: "TE_INVALID_ARGUMENT",               // Invalid argument (e.g. missing or wrong param)
	11005: "TE_MAINTENANCE_MODE",               // System maintenance mode
	11011: "TE_REDUCE_ONLY_ABORT",              // reduce-only order aborted / not allowed
	11012: "TE_REPLACE_TO_INVALID_QTY",         // Invalid quantity in order
	11013: "TE_REPLACE_TO_INVALID_PRICE",       // Invalid price in order
	11014: "TE_REPLACE_TO_INVALID_LEVERAGE",    // Invalid leverage
	11015: "TE_PRICE_TOO_SMALL",                // Price below minimum increment / tick size
	11016: "TE_PRICE_TOO_LARGE",                // Price too large
	11017: "TE_QTY_TOO_SMALL",                  // Quantity below minimum
	11018: "TE_QTY_TOO_LARGE",                  // Quantity above maximum
	11019: "TE_VALUE_TOO_SMALL",                // Value (price × qty) too small
	11020: "TE_VALUE_TOO_LARGE",                // Value too large
	11021: "TE_TOTAL_ORDER_VALUE_TOO_LARGE",    // Total order value exceeds limit
	11022: "TE_STOP_PRICE_INVALID",             // Invalid stop price for stop orders
	11037: "TE_USER_NOT_EXIST",                 // User account does not exist or is disabled
	11040: "TE_MARGIN_ACCOUNT_NOT_EXIST",       // Margin account not exist
	11041: "TE_MARGIN_ACCOUNT_FROZEN",          // Margin account frozen
	11050: "TE_RISK_LIMIT_EXCEEDED",            // Risk limit exceeded
	11051: "TE_INSUFFICIENT_BALANCE",           // Not enough balance
	11052: "TE_INSUFFICIENT_MARGIN",            // Not enough margin
	11060: "TE_POSITION_MISMATCH",              // Position mismatch error
	11061: "TE_POSITION_MARGIN_INVALID",        // Position margin invalid
	11062: "TE_POSITION_NOT_EXIST",             // Position not exist
	11063: "TE_TPSL_TOO_SMALL",                 // Take-Profit / Stop-Loss too small
	11064: "TE_TPSL_TOO_LARGE",                 // Take-Profit / Stop-Loss too large
	11065: "TE_TPSL_INVALID_TYPE",              // Invalid TP/SL type
	11066: "TE_ORDER_UNSUPPORTED",              // Unsupported order type
	11067: "TE_ORDER_DISABLED",                 // Order disabled for this symbol
	11070: "TE_MARKET_CLOSED",                  // Market closed
	11071: "TE_RESTRICTED_REGION",              // Region restricted
	11081: "TE_CLIENT_ID_EXIST",                // Duplicate client order ID
	11082: "TE_CLIENT_ID_INVALID",              // Invalid client order ID
	11100: "TE_TOO_MANY_ORDERS",                // Too many outstanding orders
	11101: "TE_TOO_MANY_ORDERS_PER_SIDE",       // Too many orders per side
	11102: "TE_TOO_MANY_ORDERS_PER_PRICE",      // Too many orders at same price
	11103: "TE_FUTURES_INVALID_MARGIN_ACCOUNT", // Invalid margin account for futures
	11104: "TE_FUTURES_INVALID_POSITION",       // Invalid position for futures
	11120: "TE_CONTRACT_NOT_FOUND",             // Contract (symbol) not found
	11121: "TE_CONTRACT_NOT_ALLOWED",           // Contract not allowed
}

// GetErrorMsg returns a human-readable message for a given Phemex error code.
// If the code is unknown, returns a generic message including the code.
func GetErrorMsg(code int) string {
	if msg, ok := PhemexErrorCodes[code]; ok {
		return msg
	}
	return fmt.Sprintf("UNKNOWN_PHEMEX_ERROR_%d", code)
}
