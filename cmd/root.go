// Copyright © 2016 Ryan Eschinger <ryanesc@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/ryane/takethe53/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "takethe53",
	Short: "Creates Route53 records.",
	Long:  `Creates Route53 records.`,
	Run: func(cmd *cobra.Command, args []string) {
		server.Run(viper.GetString("address"))
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.takethe53.yaml)")

	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose logging")
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))

	RootCmd.PersistentFlags().StringP("log-format", "f", "text", "log format. text|json")
	viper.BindPFlag("log-format", RootCmd.PersistentFlags().Lookup("log-format"))

	RootCmd.Flags().String("address", ":9053", "the address to listen on")
	viper.BindPFlag("address", RootCmd.Flags().Lookup("address"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".takethe53") // name of config file (without extension)
	viper.AddConfigPath("$HOME")      // adding home directory as first search path
	viper.SetEnvPrefix("takethe53")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	readConfig := false
	if err := viper.ReadInConfig(); err == nil {
		readConfig = len(viper.ConfigFileUsed()) > 0
	}

	if viper.GetString("log-format") == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("debug on")
	}

	if readConfig {
		logrus.Debug("Using config file:", viper.ConfigFileUsed())
	}
}

func logger(fields logrus.Fields) *logrus.Entry {
	return logrus.WithFields(fields)
}
