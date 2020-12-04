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

package controllers_test

import (
	"context"
	"math/rand"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"

	sopssecretsv1beta1 "github.com/dhouti/sops-converter/api/v1beta1"
	controllersmocks "github.com/dhouti/sops-converter/controllers/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var currentNamespace string
var currentObjectName string

var _ = Describe("sopssecret controller", func() {
	ctx := context.Background()
	maxTimeout := 5
	var mockedDecrytor *controllersmocks.DecryptorMock

	BeforeEach(func() {
		currentNamespace = getRandomString()
		currentObjectName = getRandomString()
		createNamespace(currentNamespace)

		// Simple mock, just make it return the input.
		// We can validate all other behaviors this way.
		mockedDecrytor = &controllersmocks.DecryptorMock{
			DecryptFunc: func(input []byte, format string) ([]byte, error) {
				return input, nil
			},
		}
		usedReconciler.InjectDecryptor(mockedDecrytor)
	})

	AfterEach(func() {
		// Teardown the current object after every test to prevent mock collisions in the reconciler
		currentObject := getTestSopsSecret()
		err := k8sClient.Delete(ctx, currentObject)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			fetchSopsSecret := &sopssecretsv1beta1.SopsSecret{}
			return k8sClient.Get(ctx, getNamespacedName(), fetchSopsSecret)
		}, 30).Should(HaveOccurred())
	})

	Context("fail to complete reconcile", func() {
		It("Fails to parse yaml", func() {
			newSecret := getTestSopsSecret()
			newSecret.Data = "this isn't yaml, this will fail"

			err := k8sClient.Create(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))

			createdSecret := &corev1.Secret{}
			Consistently(func() error {
				return k8sClient.Get(ctx, getNamespacedName(), createdSecret)
			}, maxTimeout).Should(HaveOccurred())
		})
	})

	Context("decrypts secrets successfuly", func() {
		It("decrypts a simple secret", func() {
			newSecret := getTestSopsSecret()
			newSecret.Data = "test: value"

			err := k8sClient.Create(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))

			createdSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, getNamespacedName(), createdSecret)
			}, maxTimeout).Should(Not(HaveOccurred()))

			Expect(createdSecret.Data["test"]).To(Equal([]byte("value")))
		})
	})

	Context("Annotation behaviors", func() {
		It("Reconcile short-circuits on match", func() {
			newSecret := getTestSopsSecret()
			newSecret.Data = "annotation: test"

			err := k8sClient.Create(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))

			createdSecretKey := getNamespacedName()
			createdSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, createdSecretKey, createdSecret)
			}, maxTimeout).Should(Not(HaveOccurred()))

			Expect(createdSecret.Data["annotation"]).To(Equal([]byte("test")))

			createdSecret.ObjectMeta.Annotations["arbitrary-update"] = "true"
			err = k8sClient.Update(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			createdSecret.ObjectMeta.Annotations["arbitrary-update"] = "false"
			err = k8sClient.Update(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Consistently(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))
		})

		It("updates the secret when sopssecret is updated", func() {
			newSecret := getTestSopsSecret()
			newSecret.Data = "secret: update"

			err := k8sClient.Create(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))

			createdSecretKey := getNamespacedName()
			createdSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, createdSecretKey, createdSecret)
			}, maxTimeout).Should(Not(HaveOccurred()))

			Expect(createdSecret.Data["secret"]).To(Equal([]byte("update")))

			newSecret.Data = "secret: test"
			err = k8sClient.Update(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() []byte {
				err = k8sClient.Get(ctx, createdSecretKey, createdSecret)
				Expect(err).ToNot(HaveOccurred())
				return createdSecret.Data["secret"]
			}, maxTimeout).Should(Equal([]byte("test")))
		})

		It("restores the secret when it is updated", func() {
			newSecret := getTestSopsSecret()
			newSecret.Data = "secret: update"

			err := k8sClient.Create(ctx, newSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				return len(mockedDecrytor.DecryptCalls())
			}, maxTimeout).Should(Equal(1))

			createdSecretKey := getNamespacedName()
			createdSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, createdSecretKey, createdSecret)
			}, maxTimeout).Should(Not(HaveOccurred()))

			Expect(createdSecret.Data["secret"]).To(Equal([]byte("update")))

			createdSecret.Data["secret"] = []byte("askdjflk")
			err = k8sClient.Update(ctx, createdSecret)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() []byte {
				err = k8sClient.Get(ctx, createdSecretKey, createdSecret)
				Expect(err).ToNot(HaveOccurred())
				return createdSecret.Data["secret"]
			}, maxTimeout).Should(Equal([]byte("update")))
		})
	})
})

func getTestSopsSecret() *sopssecretsv1beta1.SopsSecret {
	return &sopssecretsv1beta1.SopsSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      currentObjectName,
			Namespace: currentNamespace,
		},
	}
}

func getNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      currentObjectName,
		Namespace: currentNamespace,
	}
}

func getRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	b := make([]rune, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func createNamespace(target string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: target,
		},
	}
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
}
