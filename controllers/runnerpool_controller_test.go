package controllers

import (
	"context"
	"time"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RunnerPool reconciler", func() {
	ctx := context.Background()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		apr := RunnerPoolReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("RunnerPool"),
			Scheme: mgr.GetScheme(),
		}
		err = apr.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		ctx.Done()
		time.Sleep(10 * time.Millisecond)
	})

	It("should create Deployment", func() {
		By("deploying RunnerPool resource")
		rp := &actionsv1alpha1.RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "runnerpool-0",
				Namespace: "test-ns",
			},
			Spec: actionsv1alpha1.RunnerPoolSpec{
				DeploymentSpec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "runnerpool-0",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "runnerpool-0",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test",
								},
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, rp)
		Expect(err).To(Succeed())
	})
})
