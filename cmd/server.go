// Copyright © Copyright 2016 Dexter Horthy
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
	"github.com/horthy/docket/server"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the docket server",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		server.Start()
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	// TODO flags for port, backend, docker, etc

}
