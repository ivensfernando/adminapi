package keys

//
//func printUsage() {
//	fmt.Println("Available commands:")
//	fmt.Println("  help                             Show this help message")
//	fmt.Println("  shutdown                         Exit the application")
//	fmt.Println("  set_key                          Set exchange Keys for the application")
//	fmt.Println("  run_on                           Turn on the application for the exchange (requires set_key)")
//	fmt.Println("  run_off                          Turn off the application for the exchange (requires set_key)")
//	fmt.Println()
//}
//
//func main() {
//	config := GetConfig()
//
//	ctx, stop := signal.NotifyContext(
//		context.Background(),
//		os.Interrupt,
//		syscall.SIGTERM,
//	)
//
//	if err := database.InitMainDB(); err != nil {
//		logger.WithError(err).Fatal("Failed to connect to database")
//	}
//
//	defer stop()
//	userExchangeRep := repository.NewUserExchangeRepository()
//
//	reader := bufio.NewScanner(os.Stdin)
//
//	reader.Buffer(make([]byte, 0, 1024), 1024*1024)
//
//	for {
//		fmt.Print("cmd> ")
//
//		if !reader.Scan() {
//			continue
//		}
//
//		line := strings.TrimSpace(reader.Text())
//		if line == "" {
//			continue
//		}
//
//		parts := strings.Split(line, " ")
//		cmd := parts[0]
//
//		switch cmd {
//
//		case "shutdown":
//			fmt.Println("Exiting CLI...")
//			return
//
//		case "help":
//			printUsage()
//
//		case "set_key":
//			if len(parts) < 3 {
//				printUsage()
//				continue
//			}
//			userID, key, secret, orderSizePercent := parts[1], parts[2], parts[3], parts[4]
//
//			encryptKey, err := security.EncryptString(key)
//			if err != nil {
//				logger.Error(err, "Failed to encrypt key")
//				continue
//			}
//
//			encryptSecret, err := security.EncryptString(secret)
//			if err != nil {
//				logger.Error(err, "Failed to encrypt secret")
//				continue
//			}
//
//			intOrderSizePercent, err := strconv.Atoi(orderSizePercent)
//			if err != nil {
//				logger.Error(err, "Failed to convert orderSizePercent to integer")
//				continue
//			}
//
//			f := &model.UserExchange{
//				ExchangeID:       config.ExchangeID,
//				//UserID:           userID,
//				APIKeyHash:       encryptKey,
//				APISecretHash:    encryptSecret,
//				OrderSizePercent: intOrderSizePercent,
//				RunOnServer:      config.RunOnServer,
//			}
//
//			if err := userExchangeRep.Upsert(ctx, f); err != nil {
//				logger.Error(err, "Failed to upsert user exchange")
//			}
//			//printUsage()
//
//		case "run_on":
//			if len(parts) < 1 {
//				printUsage()
//				continue
//
//			}
//			userID := parts[1]
//
//			userExchange, err := userExchangeRep.GetByUserAndExchange(ctx, userID, 1)
//			if err != nil {
//				logger.Error(err, "Failed to get user exchange")
//				continue
//			}
//
//			if userExchange == nil {
//				fmt.Println("No key set for exchange 1")
//				continue
//			}
//			if userExchange.APIKeyHash == "" || userExchange.APISecretHash == "" {
//				fmt.Println("No key set for exchange 1")
//				continue
//			}
//
//			userExchange.RunOnServer = true
//
//			if err := userExchangeRep.Update(ctx, userExchange); err != nil {
//				logger.Error(err, "Failed to upsert user exchange")
//			}
//			//printUsage()
//
//		case "run_off":
//			if len(parts) < 1 {
//				printUsage()
//				continue
//			}
//			userID := parts[1]
//
//			userExchange, err := userExchangeRep.GetByUserAndExchange(ctx, userID, 1)
//			if err != nil {
//				logger.Error(err, "Failed to get user exchange")
//				continue
//			}
//			if userExchange == nil {
//				fmt.Println("No key set for exchange 1")
//				continue
//			}
//			if userExchange.APIKeyHash == "" || userExchange.APISecretHash == "" {
//				fmt.Println("No key set for exchange 1")
//				continue
//			}
//
//			userExchange.RunOnServer = false
//
//			if err := userExchangeRep.Update(ctx, userExchange); err != nil {
//				logger.Error(err, "Failed to upsert user exchange")
//			}
//
//		default:
//			fmt.Println("Unknown command:", cmd)
//			printUsage()
//		}
//	}
//
//}
