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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"

	secretsv1beta1 "github.com/dhouti/sops-converter/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:                "convert",
	Short:              "Converts a kubernetes Secret file to a SopsSecret.",
	Long:               ``,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("must provide args")
		}

		targetFile, err := ioutil.ReadFile(args[0])
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New("first arg must be a target filename")
			}
			return err
		}

		// Convert manifest to runtime.Object to see if it's a secret
		m, _, err := scheme.Codecs.UniversalDeserializer().Decode(targetFile, nil, nil)
		if err != nil {
			return err
		}

		// Assert that object is Secret, if not exit
		secret, ok := m.(*corev1.Secret)
		if !ok {
			return errors.New("file is not a Secret")
		}

		tmpSecretData := make(map[string]string)
		for k, v := range secret.Data {
			tmpSecretData[k] = string(v)
		}

		// Merge stringData into Data
		for k, v := range secret.StringData {
			tmpSecretData[k] = v
		}

		secretData, err := yaml.Marshal(tmpSecretData)
		if err != nil {
			return err
		}

		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return err
		}
		defer tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		bytes.NewReader(secretData).WriteTo(tmpfile)
		tmpfile.Sync()

		// Catch stdout in buffer
		sopsStdout := &bytes.Buffer{}

		// run sops encrypt directly
		sopsCommandArgs := append([]string{"--encrypt", "--output-type", "yaml"}, args[1:]...)
		sopsCommandArgs = append(sopsCommandArgs, tmpfile.Name())
		sopsCommand := exec.Command("sops", sopsCommandArgs...)
		sopsCommand.Stdout = sopsStdout
		err = sopsCommand.Run()
		if err != nil {
			return err
		}

		generatedSopsSecret := &secretsv1beta1.SopsSecret{}
		generatedSopsSecret.Type = secret.Type
		generatedSopsSecret.ObjectMeta = secret.ObjectMeta
		generatedSopsSecret.Spec.Template.Annotations = secret.ObjectMeta.Annotations
		generatedSopsSecret.Spec.Template.Labels = secret.ObjectMeta.Labels
		generatedSopsSecret.Data = sopsStdout.String()

		// Set the GVK or YAMLPrinter doesn't work
		gvk := schema.GroupVersionKind{
			Group:   "secrets.dhouti.dev",
			Version: "v1beta1",
			Kind:    "SopsSecret",
		}
		generatedSopsSecret.GetObjectKind().SetGroupVersionKind(gvk)
		yamlPrinter := printers.YAMLPrinter{}
		err = yamlPrinter.PrintObj(generatedSopsSecret, os.Stdout)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
