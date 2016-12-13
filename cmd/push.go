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
	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push one or more allocations from a Yaml file",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		return NewCli(cmd, args).Push()
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().String("host", "http://localhost:3000", "The host to use")
}
