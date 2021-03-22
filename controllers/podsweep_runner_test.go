package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("PodSweeper runner", func() {
	ctx := context.Background()
	organizationName := "podsweep-org"
	namespace := "podsweep-ns"
	interval := time.Second

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		sweeper := NewPodSweeper(
			mgr.GetClient(),
			ctrl.Log.WithName("actions-token-updator"),
			interval,
			organizationName,
		)
		err = mgr.Add(sweeper)
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

		err := k8sClient.Create(ctx, &pod)
		Expect(err).ShouldNot(HaveOccurred())

		By("cofirming Pod is deleted eventually")
		Eventually(func() error {
			return k8sClient.Get(
				ctx,
				types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
				&corev1.Pod{},
			)
		}, 10*time.Second).Should(HaveOccurred())
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
			err := k8sClient.Create(ctx, &pod)
			Expect(err).ShouldNot(HaveOccurred())

			By("cofirming Pod is not deleted")
			time.Sleep(5 * time.Second)
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
				&corev1.Pod{},
			)
			Expect(err).ShouldNot(HaveOccurred())

			By("cofirming Pod is deleted")
			err = k8sClient.Delete(ctx, &pod)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
					&corev1.Pod{},
				)
			}, 10*time.Second).Should(HaveOccurred())
		}
	})
})
