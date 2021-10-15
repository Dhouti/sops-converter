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
	"fmt"
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
			return errors.New("no filename given")
		}
		if len(args) > 1 {
			return errors.New("too many args")
		}
		targetFile, err := ioutil.ReadFile(args[0])
		if err != nil {
			if os.IsNotExist(err) {
				// Create a new SopsSecret if not exists?
				return nil
			}
			return err
		}

		var allDocuments []yaml.MapSlice
		// Parse out multiple objects
		var originalYaml yaml.MapSlice
		decoder := yaml.NewDecoder(bytes.NewReader(targetFile))
		for decoder.Decode(&originalYaml) == nil {
			allDocuments = append(allDocuments, originalYaml)
		}

		allObjects := map[int]*secretsv1beta1.SopsSecret{}
		for index, document := range allDocuments {
			// Convert back to yaml to parse again.
			documentBytes, err := yaml.Marshal(document)
			if err != nil {
				return err
			}

			// Convert manifest to runtime.Object to see if it's a SopsSecret
			m, _, err := scheme.Codecs.UniversalDeserializer().Decode(documentBytes, nil, nil)
			if err != nil {
				continue
			}

			// Assert that object is SopsSecret, if not exit
			sopsSecret, ok := m.(*secretsv1beta1.SopsSecret)
			if !ok {
				// Not a SopsSecret, skip
				continue
			}
			allObjects[index] = sopsSecret
		}

		if len(allObjects) == 0 {
			return errors.New("no SopsSecret objects found")
		}

		var targetIndex int
		if len(allObjects) > 1 {
			fmt.Printf("Found %v SopsSecret objects:\n", len(allObjects))
			fmt.Println("[index] name/namespace")
			for index, obj := range allObjects {
				fmt.Printf("[%v]: %s/%s\n", index, obj.Name, obj.Namespace)
			}
			fmt.Println("Enter the index of the SopsSecret you'd like to edit: ")
			fmt.Scanln(&targetIndex)
		}
		targetYamlMap := allDocuments[targetIndex]

		// Open a temporary file.
		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return err
		}
		sopsSecret := allObjects[targetIndex]

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
		for index, item := range targetYamlMap {
			keyString, ok := item.Key.(string)
			if ok && keyString == "data" {
				item.Value = string(tmpfileContents)
				targetYamlMap[index] = item
				break
			}
		}

		allDocuments[targetIndex] = targetYamlMap
		var outBuffer bytes.Buffer
		for _, document := range allDocuments {
			if document == nil {
				continue
			}
			out, err := yaml.Marshal(document)
			if err != nil {
				return err
			}
			outBuffer.Write(out)
			outBuffer.Write([]byte("---\n"))
		}

		err = ioutil.WriteFile(args[0], outBuffer.Bytes(), finfo.Mode())
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
