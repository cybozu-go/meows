package controllers

import (
	"context"
	"errors"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RunnerPool reconciler", func() {
	organizationName := "runnerpool-org"
	repositoryNames := []string{"runnerpool-repo-1", "runnerpool-repo-2"}
	namespace := "runnerpool-ns"
	runnerPoolName := "runnerpool-1"
	deploymentName := "runnerpool-1"
	slackAgentServiceName := "slack-agent"

	ctx := context.Background()
	var mgrCtx context.Context
	var mgrCancel context.CancelFunc

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		r := NewRunnerPoolReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("RunnerPool"),
			mgr.GetScheme(),
			repositoryNames,
			organizationName,
		)
		Expect(r.SetupWithManager(mgr)).To(Succeed())

		mgrCtx, mgrCancel = context.WithCancel(context.Background())
		go func() {
			err := mgr.Start(mgrCtx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(time.Second)
	})

	AfterEach(func() {
		mgrCancel()
		time.Sleep(500 * time.Millisecond)
	})

	It("should create Namespace", func() {
		createNamespaces(ctx, namespace)
	})

	It("should create Deployment with a service name of slack agent", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPoolTemplate(runnerPoolName, namespace)
		rp.Spec.RepositoryName = repositoryNames[0]
		rp.Spec.SlackAgentServiceName = &slackAgentServiceName
		rp.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  constants.RunnerContainerName,
				Image: "sample:latest",
			},
		}
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("wating the RunnerPool become Bound")
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if !rp.Status.Bound {
				return errors.New(`status "bound" should be true`)
			}
			return nil
		}).Should(Succeed())

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, d)).To(Succeed())

		By("confirming the Deployment's manifests")
		Expect(d.Labels[constants.RunnerOrgLabelKey]).To(Equal(organizationName))
		Expect(d.Labels[constants.RunnerRepoLabelKey]).To(Equal(repositoryNames[0]))
		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))
		c := d.Spec.Template.Spec.Containers[0]
		Expect(c.Env).To(HaveLen(6))
		Expect(c.Env[0].Name).To(Equal(constants.PodNameEnvName))
		Expect(c.Env[0].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.name"))
		Expect(c.Env[1].Name).To(Equal(constants.PodNamespaceEnvName))
		Expect(c.Env[1].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.namespace"))
		Expect(c.Env[2].Name).To(Equal(constants.RunnerOrgEnvName))
		Expect(c.Env[2].Value).To(Equal(organizationName))
		Expect(c.Env[3].Name).To(Equal(constants.RunnerRepoEnvName))
		Expect(c.Env[3].Value).To(Equal(repositoryNames[0]))
		Expect(c.Env[4].Name).To(Equal(constants.RunnerPoolNameEnvName))
		Expect(c.Env[4].Value).To(Equal(runnerPoolName))
		Expect(c.Env[5].Name).To(Equal(constants.SlackAgentEnvName))
		Expect(c.Env[5].Value).To(Equal(slackAgentServiceName))

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)

		By("wating the Deployment is deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &appsv1.Deployment{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	It("should create Deployment without a service name of slack agent", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPoolTemplate(runnerPoolName, namespace)
		rp.Spec.RepositoryName = repositoryNames[1]
		rp.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  constants.RunnerContainerName,
				Image: "sample:latest",
			},
		}
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("wating the RunnerPool become Bound")
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if !rp.Status.Bound {
				return errors.New(`status "bound" should be true`)
			}
			return nil
		}).Should(Succeed())

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, d)).To(Succeed())

		By("confirming the Deployment's manifests")
		Expect(d.Labels[constants.RunnerOrgLabelKey]).To(Equal(organizationName))
		Expect(d.Labels[constants.RunnerRepoLabelKey]).To(Equal(repositoryNames[1]))
		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))
		c := d.Spec.Template.Spec.Containers[0]
		Expect(c.Env).To(HaveLen(5))
		Expect(c.Env[0].Name).To(Equal(constants.PodNameEnvName))
		Expect(c.Env[0].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.name"))
		Expect(c.Env[1].Name).To(Equal(constants.PodNamespaceEnvName))
		Expect(c.Env[1].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.namespace"))
		Expect(c.Env[2].Name).To(Equal(constants.RunnerOrgEnvName))
		Expect(c.Env[2].Value).To(Equal(organizationName))
		Expect(c.Env[3].Name).To(Equal(constants.RunnerRepoEnvName))
		Expect(c.Env[3].Value).To(Equal(repositoryNames[1]))
		Expect(c.Env[4].Name).To(Equal(constants.RunnerPoolNameEnvName))
		Expect(c.Env[4].Value).To(Equal(runnerPoolName))

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)

		By("wating the Deployment is deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &appsv1.Deployment{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	It("should not create Deployment without the container with the required name", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPoolTemplate(runnerPoolName, namespace)
		rp.Spec.RepositoryName = repositoryNames[0]
		rp.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  "bad-name",
				Image: "sample:latest",
			},
		}
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("confirming the Deployment is not created")
		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &appsv1.Deployment{})
		}).ShouldNot(Succeed())

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)
	})

	It("should not create Deployment with an invalid repository name", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPoolTemplate(runnerPoolName, namespace)
		rp.Spec.RepositoryName = "bad-runnerpool-repo"
		rp.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  constants.RunnerContainerName,
				Image: "sample:latest",
			},
		}
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("confirming the Deployment is not created")
		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &appsv1.Deployment{})
		}).ShouldNot(Succeed())

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)
	})
})

func makeRunnerPoolTemplate(name, namespace string) *actionsv1alpha1.RunnerPool {
	return &actionsv1alpha1.RunnerPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			// Add a finalizer manually, because a webhook is not working in this test.
			Finalizers: []string{constants.RunnerPoolFinalizer},
		},
		Spec: actionsv1alpha1.RunnerPoolSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: actionsv1alpha1.PodTemplateSpec{
				ObjectMeta: actionsv1alpha1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
			},
		},
	}
}

func deleteRunnerPool(ctx context.Context, name, namespace string) {
	rp := &actionsv1alpha1.RunnerPool{}
	rp.Name = name
	rp.Namespace = namespace
	ExpectWithOffset(1, k8sClient.Delete(ctx, rp)).To(Succeed())
	EventuallyWithOffset(1, func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &actionsv1alpha1.RunnerPool{})
		return apierrors.IsNotFound(err)
	}).Should(BeTrue())
}
