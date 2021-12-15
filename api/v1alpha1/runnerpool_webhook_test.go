package v1alpha1

import (
	"context"

	constants "github.com/cybozu-go/meows"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeRunnerPoolTemplate(name, namespace string) *RunnerPool {
	return &RunnerPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func deleteRunnerPools(ctx context.Context, namespace string) {
	rpList := &RunnerPoolList{}
	k8sClient.List(ctx, rpList, client.InNamespace(namespace))
	for i := range rpList.Items {
		rp := rpList.Items[i].DeepCopy()

		// Remove finalizer
		rp.Finalizers = nil
		Expect(k8sClient.Update(ctx, rp)).To(Succeed())

		Expect(k8sClient.Delete(ctx, rp)).To(Succeed())
		nsn := types.NamespacedName{Name: rp.Name, Namespace: rp.Namespace}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nsn, &RunnerPool{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	}
}

var _ = Describe("validate RunnerPool webhook with ", func() {
	name := "runnerpool-test"
	namespace := "default"
	ctx := context.Background()

	AfterEach(func() {
		deleteRunnerPools(ctx, namespace)
	})

	It("should allow creating RunnerPool with Repository", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		rp.Spec.Template.Env = []corev1.EnvVar{
			{
				Name:  "GOOD_ENV",
				Value: "GOOD!",
			},
		}
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("checking default values")
		Expect(rp.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(rp.ObjectMeta.Finalizers[0]).To(Equal(constants.RunnerPoolFinalizer))
		Expect(rp.Spec.Replicas).To(BeNumerically("==", 1))
		Expect(rp.Spec.MaxRunnerPods).To(BeNumerically("==", 0))
		Expect(rp.Spec.RecreateDeadline).To(Equal("24h"))
		Expect(rp.Spec.Template.ServiceAccountName).To(Equal("default"))
	})

	It("should allow creating RunnerPool with Organization", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Organization = "test-org"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("checking default values")
		Expect(rp.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(rp.ObjectMeta.Finalizers[0]).To(Equal(constants.RunnerPoolFinalizer))
		Expect(rp.Spec.Replicas).To(BeNumerically("==", 1))
		Expect(rp.Spec.MaxRunnerPods).To(BeNumerically("==", 0))
		Expect(rp.Spec.RecreateDeadline).To(Equal("24h"))
		Expect(rp.Spec.Template.ServiceAccountName).To(Equal("default"))
	})

	It("should deny creating RunnerPool with neither Repository nor Organization", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		Expect(k8sClient.Create(ctx, rp)).NotTo(Succeed())
	})

	It("should deny creating RunnerPool with both Repository and Organization", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		rp.Spec.Organization = "test-org"
		Expect(k8sClient.Create(ctx, rp)).NotTo(Succeed())
	})

	It("should deny updating RunnerPool if Repository is changed", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo1"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		rp.Spec.Repository = "test-org/test-repo2"
		Expect(k8sClient.Update(ctx, rp)).NotTo(Succeed())
	})

	It("should deny updating RunnerPool if Organization is changed", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Organization = "test-org1"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		rp.Spec.Organization = "test-org2"
		Expect(k8sClient.Update(ctx, rp)).NotTo(Succeed())
	})

	It("should allow creating RunnerPool when Replicas == MaxRunnerPods", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		rp.Spec.Replicas = 2
		rp.Spec.MaxRunnerPods = 2
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())
	})

	It("should allow creating RunnerPool when Replicas <= MaxRunnerPods", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		rp.Spec.Replicas = 2
		rp.Spec.MaxRunnerPods = 3
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())
	})

	It("should deny creating RunnerPool when Replicas > MaxRunnerPods", func() {
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		rp.Spec.Replicas = 3
		rp.Spec.MaxRunnerPods = 2
		Expect(k8sClient.Create(ctx, rp)).NotTo(Succeed())
	})

	It("should deny updating RunnerPool when Replicas > MaxRunnerPods", func() {
		By("creating RunnerPool")
		rp := makeRunnerPoolTemplate(name, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("updating RunnerPool")
		rp.Spec.Replicas = 3
		rp.Spec.MaxRunnerPods = 2
		Expect(k8sClient.Update(ctx, rp)).NotTo(Succeed())
	})

	It("should deny creating or updating RunnerPool with reserved environment variables", func() {
		testCases := []string{
			constants.PodNameEnvName,
			constants.PodNamespaceEnvName,
			constants.RunnerOrgEnvName,
			constants.RunnerRepoEnvName,
			constants.RunnerPoolNameEnvName,
			constants.RunnerOptionEnvName,
		}

		for _, envName := range testCases {
			By("creating runner pool with reserved environment variables; " + envName)
			rp := makeRunnerPoolTemplate(name, namespace)
			rp.Spec.Repository = "test-org/test-repo"
			rp.Spec.Template.Env = []corev1.EnvVar{
				{
					Name:  envName,
					Value: "creating",
				},
			}
			Expect(k8sClient.Create(ctx, rp)).NotTo(Succeed())

			By("updating runner pool with reserved environment variables; " + envName)
			rp = makeRunnerPoolTemplate(name, namespace)
			rp.Spec.Repository = "test-org/test-repo"
			Expect(k8sClient.Create(ctx, rp)).To(Succeed())
			rp.Spec.Template.Env = []corev1.EnvVar{
				{
					Name:  envName,
					Value: "updating",
				},
			}
			Expect(k8sClient.Update(ctx, rp)).NotTo(Succeed())

			By("deleting the created RunnerPool; " + envName)
			deleteRunnerPools(ctx, namespace)
		}
	})
})
