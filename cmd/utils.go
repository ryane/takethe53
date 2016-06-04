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
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/briandowns/spinner"
	"github.com/ryane/takethe53/awsclient"
)

func waitForChangeSync(client *awsclient.AWSClient, change *awsclient.ChangeStatus, timeout int, fields logrus.Fields) {
	if change.Status == awsclient.ChangeStatusInSync {
		return
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
	s.Start()

	id := change.ID
	c := make(chan string, 1)

	go func() {
		for {
			time.Sleep(2 * time.Second)
			status, err := client.GetChangeStatus(id)
			if err != nil {
				logger(fields).Fatal("Error checking status: ", err)
			}
			if status.Status == awsclient.ChangeStatusInSync {
				c <- status.Status
			}
		}
	}()

	var message string
	select {
	case <-c:
		message = "Done."
	case <-time.After(time.Second * time.Duration(timeout)):
		message = fmt.Sprintf("It is taking longer than expected to synchronize the change to all Route53 DNS servers. You can check the status with the AWS CLI.\n\naws route53 get-change --id %s\n", id)
	}

	s.Stop()
	fmt.Printf(" %s\n", message)
}
