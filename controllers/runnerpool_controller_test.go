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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RunnerPool reconciler", func() {
	ctx := context.Background()
	organizationName := "runnerpool-org"
	repositoryName := "runnerpool-repo"
	namespace := "runnerpool-ns"
	slackAgentServiceName := "slack-agent"

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
			organizationName,
		)
		err = r.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(time.Second)
	})

	AfterEach(func() {
		ctx.Done()
		time.Sleep(500 * time.Millisecond)
	})

	It("should create Namespace", func() {
		By("creating namespace")
		ctx := context.Background()
		ns := &corev1.Namespace{}
		ns.Name = namespace
		err := k8sClient.Create(ctx, ns)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not create Deployment", func() {
		name := "runnerpool-0"
		By("deploying RunnerPool resource without the container with the required name")
		rp := &actionsv1alpha1.RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: actionsv1alpha1.RunnerPoolSpec{
				RepositoryName: repositoryName,
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
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "bad-name",
								Image: "sample:latest",
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, rp)
		Expect(err).To(Succeed())

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		nsn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, nsn, rp); err != nil {
				return err
			}

			return k8sClient.Get(ctx, nsn, d)
		}, 5*time.Second).ShouldNot(Succeed())

		By("deleting the created RunnerPool")
		rp = &actionsv1alpha1.RunnerPool{}
		err = k8sClient.Get(ctx, nsn, rp)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Delete(ctx, rp)
		Expect(err).NotTo(HaveOccurred())

		By("deploying RunnerPool resource with an invalid repository name")
		rp = &actionsv1alpha1.RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: actionsv1alpha1.RunnerPoolSpec{
				RepositoryName: "bad-runnerpool-repo",
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
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  constants.RunnerContainerName,
								Image: "sample:latest",
							},
						},
					},
				},
			},
		}
		err = k8sClient.Create(ctx, rp)
		Expect(err).To(Succeed())

		By("getting the created Deployment")
		d = new(appsv1.Deployment)
		nsn = types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, nsn, rp); err != nil {
				return err
			}

			return k8sClient.Get(ctx, nsn, d)
		}, 5*time.Second).ShouldNot(Succeed())

		By("deleting the created RunnerPool")
		rp = &actionsv1alpha1.RunnerPool{}
		err = k8sClient.Get(ctx, nsn, rp)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Delete(ctx, rp)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create Deployment", func() {
		name := "runnerpool-1"
		By("deploying RunnerPool resource")
		rp := &actionsv1alpha1.RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: actionsv1alpha1.RunnerPoolSpec{
				RepositoryName:        repositoryName,
				SlackAgentServiceName: &slackAgentServiceName,
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
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  constants.RunnerContainerName,
								Image: "sample:latest",
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, rp)
		Expect(err).To(Succeed())

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		nsn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, nsn, rp); err != nil {
				return err
			}

			if err := k8sClient.Get(ctx, nsn, d); err != nil {
				return err
			}

			if !rp.Status.Bound {
				return errors.New(`status "bound" should be true`)
			}
			return nil
		}, 5*time.Second).Should(Succeed())

		Expect(d.Labels[constants.RunnerOrgLabelKey]).To(Equal(organizationName))
		Expect(d.Labels[constants.RunnerRepoLabelKey]).To(Equal(repositoryName))
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
		Expect(c.Env[3].Value).To(Equal(repositoryName))
		Expect(c.Env[4].Name).To(Equal(constants.SlackAgentEnvName))
		Expect(c.Env[4].Value).To(Equal(slackAgentServiceName))
	})
})
