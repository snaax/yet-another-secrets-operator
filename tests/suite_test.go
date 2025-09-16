package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = secretsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create a context for test execution
	ctx, cancel = context.WithCancel(context.TODO())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("ASecret Controller", func() {
	const (
		// Test namespace and names
		testNamespace = "default"
		asecretName   = "test-asecret"
		targetSecret  = "test-k8s-secret"
		awsSecretPath = "/test/secrets"
		timeout       = time.Second * 10
		interval      = time.Millisecond * 250
	)

	// Test setup: create AGenerator for password generation
	Context("When creating an AGenerator", func() {
		It("Should create successfully", func() {
			generator := &secretsv1alpha1.AGenerator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-generator",
					Namespace: testNamespace,
				},
				Spec: secretsv1alpha1.AGeneratorSpec{
					Length:              12,
					IncludeUppercase:    true,
					IncludeLowercase:    true,
					IncludeNumbers:      true,
					IncludeSpecialChars: true,
					SpecialChars:        "!@#$%^&*",
				},
			}

			// Create the AGenerator
			Expect(k8sClient.Create(ctx, generator)).Should(Succeed())

			// Verify it was created
			generatorLookupKey := types.NamespacedName{Name: "test-generator", Namespace: testNamespace}
			createdGenerator := &secretsv1alpha1.AGenerator{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, generatorLookupKey, createdGenerator)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	// Test creating an ASecret and verify the target K8s secret gets created
	Context("When creating an ASecret", func() {
		It("Should create the target K8s Secret", func() {
			By("Creating a new ASecret")
			asecret := &secretsv1alpha1.ASecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      asecretName,
					Namespace: testNamespace,
				},
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretName: targetSecret,
					AwsSecretPath:    awsSecretPath,
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {
							Value: "admin",
						},
						"password": {
							GeneratorRef: &secretsv1alpha1.GeneratorReference{
								Name: "test-generator",
							},
						},
					},
				},
			}

			// Create the ASecret
			Expect(k8sClient.Create(ctx, asecret)).Should(Succeed())

			// Look up the ASecret
			asecretLookupKey := types.NamespacedName{Name: asecretName, Namespace: testNamespace}
			createdASecret := &secretsv1alpha1.ASecret{}

			// Make sure it gets created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, asecretLookupKey, createdASecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Check if the target K8s Secret is created
			By("Checking if the target K8s Secret was created")
			secretLookupKey := types.NamespacedName{Name: targetSecret, Namespace: testNamespace}
			createdSecret := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify the secret data
			Expect(createdSecret.Data).To(HaveKey("username"))
			Expect(string(createdSecret.Data["username"])).To(Equal("admin"))

			// Password should be generated
			Expect(createdSecret.Data).To(HaveKey("password"))
			Expect(len(string(createdSecret.Data["password"]))).To(Equal(12)) // Length from generator spec
		})
	})

	// Test updating an ASecret
	Context("When updating an ASecret", func() {
		It("Should update the target K8s Secret", func() {
			// Get the existing ASecret
			asecretLookupKey := types.NamespacedName{Name: asecretName, Namespace: testNamespace}
			existingASecret := &secretsv1alpha1.ASecret{}

			Expect(k8sClient.Get(ctx, asecretLookupKey, existingASecret)).To(Succeed())

			// Update the ASecret with a new field
			existingASecret.Spec.Data["api-key"] = secretsv1alpha1.DataSource{
				Value: "abc123",
			}

			// Update the ASecret
			Expect(k8sClient.Update(ctx, existingASecret)).To(Succeed())

			// Check if the target K8s Secret is updated with the new field
			secretLookupKey := types.NamespacedName{Name: targetSecret, Namespace: testNamespace}
			updatedSecret := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, updatedSecret)
				if err != nil {
					return false
				}
				return updatedSecret.Data != nil && len(updatedSecret.Data["api-key"]) > 0
			}, timeout, interval).Should(BeTrue())

			// Verify the new field
			Expect(string(updatedSecret.Data["api-key"])).To(Equal("abc123"))
		})
	})

	// Test removing a field from an ASecret
	Context("When removing a field from an ASecret", func() {
		It("Should remove the field from the target K8s Secret", func() {
			// Get the existing ASecret
			asecretLookupKey := types.NamespacedName{Name: asecretName, Namespace: testNamespace}
			existingASecret := &secretsv1alpha1.ASecret{}

			Expect(k8sClient.Get(ctx, asecretLookupKey, existingASecret)).To(Succeed())

			// Remove the api-key field
			delete(existingASecret.Spec.Data, "api-key")

			// Update the ASecret
			Expect(k8sClient.Update(ctx, existingASecret)).To(Succeed())

			// Check if the target K8s Secret has the field removed
			secretLookupKey := types.NamespacedName{Name: targetSecret, Namespace: testNamespace}
			updatedSecret := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, updatedSecret)
				if err != nil {
					return false
				}
				_, exists := updatedSecret.Data["api-key"]
				return !exists
			}, timeout, interval).Should(BeTrue())

			// Verify the field is gone
			Expect(updatedSecret.Data).NotTo(HaveKey("api-key"))
		})
	})

	// Test deleting an ASecret
	Context("When deleting an ASecret", func() {
		It("Should delete the target K8s Secret if configured", func() {
			// Get the existing ASecret
			asecretLookupKey := types.NamespacedName{Name: asecretName, Namespace: testNamespace}
			existingASecret := &secretsv1alpha1.ASecret{}

			Expect(k8sClient.Get(ctx, asecretLookupKey, existingASecret)).To(Succeed())

			// Set DeletePolicy to "Delete" to ensure the target secret is removed
			// existingASecret.Spec.DeletePolicy = "Delete"
			Expect(k8sClient.Update(ctx, existingASecret)).To(Succeed())

			// Delete the ASecret
			Expect(k8sClient.Delete(ctx, existingASecret)).To(Succeed())

			// Check if the ASecret was deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, asecretLookupKey, &secretsv1alpha1.ASecret{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			// Check if the target K8s Secret was also deleted
			secretLookupKey := types.NamespacedName{Name: targetSecret, Namespace: testNamespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, &corev1.Secret{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
