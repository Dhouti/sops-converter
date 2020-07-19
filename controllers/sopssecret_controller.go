/*


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
	"encoding/json"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretsv1beta1 "github.com/dhouti/sops-converter/api/v1beta1"
	sops "go.mozilla.org/sops/v3/decrypt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SopsSecretReconciler reconciles a SopsSecret object
type SopsSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=secrets.dhouti.dev,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.dhouti.dev,resources=sopssecrets/status,verbs=get;update;patch

func (r *SopsSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("sopssecret", req.NamespacedName)

	obj := &secretsv1beta1.SopsSecret{}
	err := r.Get(context.TODO(), req.NamespacedName, obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Decrypt the Data field using Sops
	unencryptedData, err := sops.Data([]byte(obj.Data), "yaml")
	if err != nil {
		return ctrl.Result{}, err
	}

	// Convert decryted secret into map[string]string, sadly cannot unmarshal directly into []byte
	secretDataStrings := make(map[string]string)
	err = json.Unmarshal(unencryptedData, &secretDataStrings)
	if err != nil {
		return ctrl.Result{}, err
	}

	// meh
	generatedSecretData := make(map[string][]byte)
	for k, v := range secretDataStrings {
		generatedSecretData[k] = []byte(v)
	}

	generatedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.Name,
			Namespace: obj.Namespace,
		},
		// TypeMeta must be specified for server side apply.
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: corev1.SecretTypeOpaque,
		Data: generatedSecretData,
	}

	err = r.Patch(context.TODO(), generatedSecret, client.Apply, []client.PatchOption{client.ForceOwnership, client.FieldOwner("sopsecret-controller")}...)
	return ctrl.Result{}, err
}

func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&secretsv1beta1.SopsSecret{}).Owns(&corev1.Secret{}).Complete(r)
}
