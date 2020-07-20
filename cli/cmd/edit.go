/*
Copyright © 2020 Rex Via  l.rex.via@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"

	secretsv1beta1 "github.com/dhouti/sops-converter/api/v1beta1"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Opens and decrypts a SopsSecret.",
	Long: `Opens a SopsSecret manifest, decrypts it,
		and loads it into the $EDITOR of your choice.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("No filename given.")
		}
		if len(args) > 1 {
			return errors.New("Too many args.")
		}
		targetFile, err := ioutil.ReadFile(args[0])
		if err != nil {
			if os.IsNotExist(err) {
				// Create a new SopsSecret if not exists?
				return nil
			}
			return err
		}

		// Store the original yaml
		var originalYaml yaml.MapSlice
		err = yaml.Unmarshal(targetFile, &originalYaml)
		if err != nil {
			return err
		}

		// Convert manifest to runtime.Object to see if it's a SopsSecret
		m, _, err := scheme.Codecs.UniversalDeserializer().Decode(targetFile, nil, nil)
		if err != nil {
			return err
		}

		// Assert that object is SopsSecret, if not exit
		sopsSecret, ok := m.(*secretsv1beta1.SopsSecret)
		if !ok {
			return errors.New("file is not a SopsSecret")
		}

		// Open a temporary file.
		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return err
		}

		defer tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		bytes.NewReader([]byte(sopsSecret.Data)).WriteTo(tmpfile)
		tmpfile.Sync()

		// Open sops editor directly
		sopsCommand := exec.Command("sops", tmpfile.Name())
		sopsCommand.Stdin = os.Stdin
		sopsCommand.Stdout = os.Stdout
		sopsCommand.Stderr = os.Stderr
		err = sopsCommand.Run()
		if err != nil {
			return err
		}

		tmpfileContents, err := ioutil.ReadFile(tmpfile.Name())
		if err != nil {
			return err
		}

		// Fetch file mode so it's not changed on write.
		finfo, err := os.Stat(args[0])
		if err != nil {
			return err
		}

		// Using yaml.MapSlice to preserve key order.
		for index, item := range originalYaml {
			keyString, ok := item.Key.(string)
			if ok && keyString == "data" {
				item.Value = string(tmpfileContents)
				originalYaml[index] = item
				break
			}
		}

		out, err := yaml.Marshal(originalYaml)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(args[0], out, finfo.Mode())
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
