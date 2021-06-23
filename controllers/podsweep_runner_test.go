package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("PodSweeper runner", func() {
	organizationName := "podsweep-org"
	namespace := "podsweep-ns"

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

		sweeper := NewPodSweeper(
			mgr.GetClient(),
			ctrl.Log.WithName("pod-sweeper"),
			time.Second,
			organizationName,
		)
		Expect(mgr.Add(sweeper)).To(Succeed())

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

	It("should delete pods", func() {
		By("creating Pod")
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample0",
				Namespace: namespace,
				Labels: map[string]string{
					constants.RunnerOrgLabelKey: organizationName,
				},
				Annotations: map[string]string{
					constants.PodDeletionTimeKey: time.Now().Add(time.Second).Format(time.RFC3339),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "sample",
						Image: "sample:latest",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, &pod)).To(Succeed())

		By("cofirming Pod is deleted eventually")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "sample0", Namespace: namespace}, &corev1.Pod{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	It("should not delete pods", func() {
		testCases := []struct {
			name  string
			input corev1.Pod
		}{
			{
				"without labels",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample1",
						Namespace: namespace,
						Annotations: map[string]string{
							constants.PodDeletionTimeKey: time.Now().UTC().Format(time.RFC3339),
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "sample",
								Image: "sample:latest",
							},
						},
					},
				},
			},
			{
				"without annotation",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample2",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "sample",
								Image: "sample:latest",
							},
						},
					},
				},
			},
		}

		for _, tt := range testCases {
			By("creating pod " + tt.name)
			pod := tt.input
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())
			nsn := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}

			By("cofirming test pod is not deleted")
			time.Sleep(5 * time.Second)
			Expect(k8sClient.Get(ctx, nsn, &corev1.Pod{})).To(Succeed())

			By("deleting test pod")
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, nsn, &corev1.Pod{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		}
	})
})
