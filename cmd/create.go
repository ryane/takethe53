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

type createParams struct {
	alias        string
	zoneName     string
	elbDnsName   string
	hostedZoneID string
}

var cParams createParams

var createCmd = &cobra.Command{
	Use:   "create <alias> <zone_name> <elb_dns_name>",
	Short: "Create or update a Route53 alias for an ELB",
	Long:  `Create or update a Route53 alias for an ELB`,
	Run: func(cmd *cobra.Command, args []string) {
		client := awsclient.New()

		if len(args) < 3 {
			cmd.Usage()
			os.Exit(1)
		}

		cParams = createParams{
			alias:      args[0],
			zoneName:   args[1],
			elbDnsName: args[2],
		}

		zone, err := client.FindZone(cParams.zoneName)
		if err != nil {
			logger(createFields()).Fatal("Error finding zone: ", err)
		}

		lb, err := client.FindLoadBalancer(cParams.elbDnsName)
		if err != nil {
			logger(createFields()).Fatal("Error finding load balancer: ", err)
		}

		cParams.hostedZoneID = lb.HostedZoneID
		change, err := client.SetAlias(zone, cParams.hostedZoneID, lb.Name, cParams.alias)
		if err != nil {
			logger(createFields()).Fatal("Error setting alias: ", err)
		}

		fmt.Print("Pending...  ")
		waitForChangeSync(client, change, 60, createFields())
	},
}

func createFields() logrus.Fields {
	return logrus.Fields{
		"op":           "create",
		"zone":         cParams.zoneName,
		"alias":        cParams.alias,
		"elbDnsName":   cParams.elbDnsName,
		"hostedZoneID": cParams.hostedZoneID,
	}
}

func init() {
	RootCmd.AddCommand(createCmd)
}
