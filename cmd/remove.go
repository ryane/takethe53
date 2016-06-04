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
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/ryane/takethe53/awsclient"
	"github.com/spf13/cobra"
)

type removeParams struct {
	alias    string
	zoneName string
}

var rParams removeParams

var removeCmd = &cobra.Command{
	Use:   "remove <alias> <zone_name>",
	Short: "Remove a Route53 alias for an ELB",
	Long:  `Remove a Route53 alias for an ELB`,
	Run: func(cmd *cobra.Command, args []string) {
		client := awsclient.New()

		if len(args) < 2 {
			cmd.Usage()
			os.Exit(1)
		}

		rParams = removeParams{
			alias:    args[0],
			zoneName: args[1],
		}

		zone, err := client.FindZone(rParams.zoneName)
		if err != nil {
			logger(removeFields()).Fatal("Error finding zone: ", err)
		}

		change, err := client.RemoveAlias(zone, rParams.alias)
		if err != nil {
			logger(removeFields()).Fatal("Error removing alias: ", err)
		}

		fmt.Printf("change status: %s\n", change.Status)
	},
}

func removeFields() logrus.Fields {
	return logrus.Fields{
		"op":    "remove",
		"zone":  rParams.zoneName,
		"alias": rParams.alias,
	}
}

func init() {
	RootCmd.AddCommand(removeCmd)
}
