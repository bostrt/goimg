package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/unrolled/logger"
)

var (
	cfg     Config
	cfgFile string
)

func main() {

	// The sole command
	var rootCmd = &cobra.Command{
		Use:   "goimg",
		Short: "",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Starting on %s...\n", cfg.bind)
			run()
		},
	}
	// Setup command line arguments and link to config file properties
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&cfg.bind, "bind", "b", "0.0.0.0:8000", "[int]:<port> to bind to")
	rootCmd.PersistentFlags().StringVarP(&cfg.data, "data", "", "./data", "path to data directory")
	rootCmd.PersistentFlags().StringVarP(&cfg.db, "db", "", "./test.db", "path to database")
	rootCmd.PersistentFlags().IntVarP(&cfg.gcInterval, "gcinterval", "", 300, "garbage collection interval in seconds")
	rootCmd.PersistentFlags().IntVarP(&cfg.gcLimit, "gclimit", "", 100, "garbage collection limit per run")
	viper.BindPFlag("bind", rootCmd.PersistentFlags().Lookup("bind"))
	viper.BindPFlag("data", rootCmd.PersistentFlags().Lookup("data"))
	viper.BindPFlag("db", rootCmd.PersistentFlags().Lookup("db"))
	viper.BindPFlag("gcinterval", rootCmd.PersistentFlags().Lookup("gcinterval"))
	viper.BindPFlag("gclimit", rootCmd.PersistentFlags().Lookup("gclimit"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func run() {
	var wg sync.WaitGroup

	// Validate directories
	if _, err := os.Stat(filepath.Dir(cfg.db)); os.IsNotExist(err) {
		fmt.Println("Database directory does not exist:", cfg.db)
		return
	} else if finfo, _ := os.Stat(cfg.db); finfo != nil && finfo.IsDir() {
		fmt.Println("Databsae location must not be a directory:", cfg.db)
		return
	}

	if finfo, err := os.Stat(cfg.data); os.IsNotExist(err) {
		fmt.Println("Data directory does not exist:", cfg.data)
		return
	} else if !finfo.IsDir() {
		fmt.Println("Data flag is not a directory:", cfg.data)
		return
	}

	fmt.Println("Opening database:", cfg.db)
	db, err := bolt.Open(cfg.db, 0600, nil)
        if err != nil {
            fmt.Println("Error opening database", err)
            return
        }
	defer db.Close()

	logger := logger.New(logger.Options{
		RemoteAddressHeaders: []string{"X-Forwarded-For"},
		OutputFlags:          log.LstdFlags,
		IgnoredRequestURIs:   []string{"/favicon.ico"},
	})
	dao := NewImageDao(db, logger)
	fs := NewFS(cfg, logger)
	gc := NewGC(db, dao, fs, &wg, logger)

	go gc.Start()

	db.Update(func(tx *bolt.Tx) error {
		// Ensure "recent" and "gc" buckets are present.
		tx.CreateBucketIfNotExists(B(RECENT_BUCKET))
		tx.CreateBucketIfNotExists(B(EXPIRATION_BUCKET))
		tx.CreateBucketIfNotExists(B(IMAGE_BUCKET))

		return nil
	})

	NewServer(dao, fs, cfg, logger).ListenAndServe()

	wg.Wait()
}

func initConfig() {
	// Read in environment variables with prefix GOIMG_
	viper.SetEnvPrefix("GOIMG")
	viper.AutomaticEnv()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Can't read config:", err)
			os.Exit(1)
		}

	}

	cfg.bind = viper.GetString("bind")
	cfg.data = viper.GetString("data")
	cfg.db = viper.GetString("db")
	cfg.gcInterval = viper.GetInt("gcinterval")
	cfg.gcLimit = viper.GetInt("gclimit")
}
