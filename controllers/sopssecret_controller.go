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

package controllers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	secretsv1beta1 "github.com/dhouti/sops-converter/api/v1beta1"
	sops "go.mozilla.org/sops/v3/decrypt"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SecretChecksumAnotation string = "secrets.dhouti.dev/secretChecksum"
const SopsChecksumAnnotation string = "secrets.dhouti.dev/sopsChecksum"
const IgnoreKeysAnnotations string = "secrets.dhouti.dev/ignoreKeys"

var _ Decryptor = &SopsDecrytor{}

//go:generate moq -out mocks/decryptor_mock.go -pkg controllers_mocks . Decryptor
type Decryptor interface {
	Decrypt([]byte, string) ([]byte, error)
}

// SopsSecretReconciler reconciles a SopsSecret object
type SopsSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Decryptor
}

type SopsDecrytor struct {
}

func (d *SopsDecrytor) Decrypt(input []byte, outFormat string) ([]byte, error) {
	return sops.Data(input, outFormat)
}

func (r *SopsSecretReconciler) InjectDecryptor(d Decryptor) {
	r.Decryptor = d
}

// +kubebuilder:rbac:groups=secrets.dhouti.dev,resources=sopssecrets,verbs="*"
// +kubebuilder:rbac:groups=secrets.dhouti.dev,resources=sopssecrets/status,verbs="*"
// +kubebuilder:rbac:groups="",resources=secrets,verbs="*"

func (r *SopsSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// If not otherwise defined, default to the real decrypt func.
	if r.Decryptor == nil {
		realDecryptor := &SopsDecrytor{}
		r.Decryptor = realDecryptor
	}
	log := r.Log.WithValues("sopssecret", req.NamespacedName)

	obj := &secretsv1beta1.SopsSecret{}
	err := r.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		log.Error(err, "failed to get sopssecret object")
		return ctrl.Result{}, err
	}

	fetchSecret := &corev1.Secret{}
	err = r.Get(ctx, req.NamespacedName, fetchSecret)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			log.Error(err, "failed to get secret object")
			return ctrl.Result{}, err
		}
	}

	// Calculate hashes of both objects to see if they are in desired state.
	secretDataBytes, err := json.Marshal(fetchSecret.Data)
	if err != nil {
		return ctrl.Result{}, err
	}

	currentSecretChecksum := hashItem(secretDataBytes)
	currentSopsChecksum := hashItem([]byte(obj.Data))

	existingSecretChecksum, hasSecretChecksum := fetchSecret.Annotations[SecretChecksumAnotation]
	existingSopsChecksum, hasSopsChecksum := fetchSecret.Annotations[SopsChecksumAnnotation]
	if (hasSecretChecksum && hasSopsChecksum) && (existingSecretChecksum == currentSecretChecksum) && (existingSopsChecksum == currentSopsChecksum) {
		log.Info("Checksums matched, skipping.")
		return ctrl.Result{}, nil
	}

	// Decrypt the Data field using Sops
	unencryptedData, err := r.Decrypt([]byte(obj.Data), "yaml")
	if err != nil {
		log.Error(err, "failed to decrypt data")
		return ctrl.Result{}, err
	}

	// Convert decryted secret into map[string]string, sadly cannot unmarshal directly into []byte
	secretDataStrings := make(map[string]string)
	err = yaml.Unmarshal(unencryptedData, &secretDataStrings)
	if err != nil {
		log.Error(err, "failed to unmarshal decrypted data")
		return ctrl.Result{}, err
	}

	// Convert map[string]string to map[string][]byte for compatibility with corev1.Secret
	generatedSecretData := make(map[string][]byte)
	for k, v := range secretDataStrings {
		generatedSecretData[k] = []byte(v)
	}

	// Add back ignored keys from live secret
	ignoredSecretKeysRaw, ok := obj.ObjectMeta.Annotations[IgnoreKeysAnnotations]
	if ok {
		ignoredKeys := strings.Split(ignoredSecretKeysRaw, ",")
		for _, key := range ignoredKeys {
			existingKey, ok := fetchSecret.Data[key]
			if !ok {
				continue
			}

			generatedSecretData[key] = existingKey
		}
	}

	// Prevents an unnecessary reconcile on new objects
	secretDataBytes, err = json.Marshal(generatedSecretData)
	if err != nil {
		return ctrl.Result{}, err
	}

	currentSecretChecksum = hashItem(secretDataBytes)

	generatedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.Name,
			Namespace: obj.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, generatedSecret, func() error {
		baseAnnotations := make(map[string]string)
		if obj.Annotations != nil {
			baseAnnotations = obj.Annotations
		}
		generatedSecret.Annotations = baseAnnotations
		generatedSecret.Annotations[SecretChecksumAnotation] = currentSecretChecksum
		generatedSecret.Annotations[SopsChecksumAnnotation] = currentSopsChecksum

		baseLabels := make(map[string]string)
		if obj.Labels != nil {
			baseLabels = obj.Labels
		}
		generatedSecret.Labels = baseLabels
		generatedSecret.Data = generatedSecretData

		generatedSecret.Type = obj.Type
		return nil
	})

	if err != nil {
		log.Error(err, "failed to apply changes to secret")
	}

	return ctrl.Result{}, err
}

func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1beta1.SopsSecret{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(
			func(o client.Object) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      o.GetName(),
							Namespace: o.GetNamespace(),
						},
					},
				}
			},
		)).
		Complete(r)
}

func hashItem(data []byte) string {
	hash := sha1.Sum(data)
	encodedHash := hex.EncodeToString(hash[:])
	return encodedHash
}
