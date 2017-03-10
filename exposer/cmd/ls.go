// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/juju/errors"
	"github.com/service-exposer/exposer/service"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "list exposed services",
}

func init() {
	RootCmd.AddCommand(lsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	lsCmd.Run = func(cmd *cobra.Command, args []string) {
		url := server_http_url() + "/api/services"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			exit(-1, errors.ErrorStack(errors.Annotatef(err, "GET %s", url)))
			os.Exit(-1)
		}
		req.Header.Set("Authorization", key)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			exit(-2, errors.ErrorStack(errors.Annotatef(err, "http.Client.Do")))
		}
		defer resp.Body.Close()

		ok := (200 <= resp.StatusCode && resp.StatusCode <= 299)
		if !ok {
			io.Copy(os.Stderr, resp.Body)
			os.Exit(1)
		}

		var result map[string]*service.Attribute
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			exit(2, errors.ErrorStack(errors.Trace(err)))
		}

		data, err := json.MarshalIndent(&result, "", "  ")
		if err != nil {
			exit(3, errors.ErrorStack(errors.Trace(err)))
		}

		os.Stdout.Write(data)
		os.Stdout.Write([]byte{'\n'})
	}
}
