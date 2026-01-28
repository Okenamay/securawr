package cli

import (
	"os"

	logger "github.com/Okenamay/securawr/internal/logger/zap"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgFile string
	log     *zap.Logger
)

// Execute - точка входа для CLI
func Execute() {
	// Инициализация логгера (пока простой вывод в stderr для CLI)
	var err error
	log, err = logger.New("info")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// rootCmd представляет базовую команду (securawr)
var rootCmd = &cobra.Command{
	Use:   "securawr",
	Short: "SecuRawr Password Manager CLI",
	Long:  `Secure client for storing passwords and binary data.`,
}

func init() {
	// Здесь определяем глобальные флаги
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.securawr.yaml)")
}
