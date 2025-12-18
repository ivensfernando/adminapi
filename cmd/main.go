package main

import (
	"fmt"
	"os"
	"strategyexecutor/cmd/executor"
	"strategyexecutor/cmd/ohlcvcrypto"
	"strategyexecutor/cmd/tv_news"
	"strategyexecutor/src/database"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var Version string

func main() {
	app := cli.NewApp()
	app.Name = "Biidin CMD"
	app.Usage = "The Biidin command line interface"

	app.Commands = []cli.Command{
		tvNewsCMD,
		executorCMD,
		ohlcvCryptoCMD,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	tvNewsCMD = cli.Command{
		Name:        "tvnews",
		Usage:       "run TV NEWS",
		Action:      tvNewsAction,
		ArgsUsage:   "",
		Flags:       []cli.Flag{},
		Description: `Run TV NEWS CMD`,
	}
	executorCMD = cli.Command{
		Name:        "executor",
		Usage:       "run Executor",
		Action:      executorAction,
		ArgsUsage:   "",
		Flags:       []cli.Flag{},
		Description: `Run Executor CMD`,
	}
	ohlcvCryptoCMD = cli.Command{
		Name:        "ohlcv_crypto",
		Usage:       "run OHLCV crypto",
		Action:      ohlcvCryptoAction,
		ArgsUsage:   "",
		Flags:       []cli.Flag{},
		Description: `Run OHLCV crypto CMD`,
	}
)

func tvNewsAction(_ *cli.Context) error {

	logrus.Info("Starting tvnews data CMD")
	logrus.WithField("cmd", "tvnews")

	tvn := &tv_news.TVNews{}
	err := tvn.Start()
	if err != nil {
		logrus.WithError(err).Error("Starting cmd")
		return err
	}

	return nil
}

func executorAction(_ *cli.Context) error {

	logrus.Info("Starting executor CMD")
	logrus.WithField("cmd", "executor")

	executorStrategy := &executor.Executor{}
	err := executorStrategy.Start()
	if err != nil {
		logrus.WithError(err).Error("Starting cmd")
		return err
	}

	return nil
}

// ohlcvCryptoAction will go get OHLCV candles for BTC/ETH
func ohlcvCryptoAction(_ *cli.Context) error {

	logrus.Info("Starting OHLCV crypto CMD")
	if err := database.InitMainDB(); err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}
	_ohlcv := &ohlcvcrypto.OHLCVCrypto{
		Log: logrus.WithField("cmd", "ohlcv_crypto"),
		DB:  database.MainDB,
	}

	err := _ohlcv.Start()
	if err != nil {
		logrus.WithError(err).Error("Starting OHLCV cmd")
		return err
	}

	return nil
}
