package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("PodSweeper runner", func() {
	organizationName := "podsweep-org"
	namespace := "podsweep-ns"

	ctx := context.Background()
	var runnerPodClient RunnerPodClientMock
	var mgrCtx context.Context
	var mgrCancel context.CancelFunc

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		runnerPodClient = NewRunnerPodClientMock("")

		sweeper := NewTestPodSweeper(
			mgr.GetClient(),
			ctrl.Log.WithName("pod-sweeper"),
			time.Second,
			organizationName,
			&runnerPodClient,
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
		testCases := []struct {
			name         string
			input        corev1.Pod
			deletionTime string
		}{
			{
				"with annotation and without API",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample0",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
						Annotations: map[string]string{
							constants.PodDeletionTimeKey: time.Now().Add(time.Second).UTC().Format(time.RFC3339),
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
				"",
			},
			{
				"with annotation and with API that return future time",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample1",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
						Annotations: map[string]string{
							constants.PodDeletionTimeKey: time.Now().Add(time.Second).UTC().Format(time.RFC3339),
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
				time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			},
			{
				"without annotation and with API",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample1",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
						Annotations: map[string]string{},
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
				time.Now().Add(1 * time.Second).UTC().Format(time.RFC3339),
			},
		}
		for _, tt := range testCases {
			runnerPodClient.deletionTime = tt.deletionTime
			pod := tt.input
			By("creating Pod" + tt.name)
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())
			nsn := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}

			By("cofirming Pod is deleted eventually")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, nsn, &corev1.Pod{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		}
	})

	It("should not delete pods", func() {
		testCases := []struct {
			name         string
			input        corev1.Pod
			deletionTime string
		}{
			{
				"without labels",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample2",
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
				"",
			},
			{
				"without annotation and without API",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample3",
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
				"",
			},
			{
				"without annotation and with API that return future time",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample4",
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
				time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			{
				"with future time annotation and without API",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample5",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
						Annotations: map[string]string{
							constants.PodDeletionTimeKey: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
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
				"",
			},
			{
				"with future time annotation and with API that return time already past",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample6",
						Namespace: namespace,
						Labels: map[string]string{
							constants.RunnerOrgLabelKey: organizationName,
						},
						Annotations: map[string]string{
							constants.PodDeletionTimeKey: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
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
				time.Now().UTC().Format(time.RFC3339),
			},
		}

		for _, tt := range testCases {
			runnerPodClient.deletionTime = tt.deletionTime
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

type RunnerPodClientMock struct {
	deletionTime string
}

func (c *RunnerPodClientMock) GetDeletionTime(ip string) (string, error) {
	return c.deletionTime, nil
}

func NewRunnerPodClientMock(deletionTime string) RunnerPodClientMock {
	return RunnerPodClientMock{
		deletionTime: deletionTime,
	}
}

func NewTestPodSweeper(
	k8sClient client.Client,
	log logr.Logger,
	interval time.Duration,
	organizationName string,
	runnerPodClient RunnerPodClient,
) manager.Runnable {
	return &PodSweeper{
		k8sClient:        k8sClient,
		log:              log,
		interval:         interval,
		organizationName: organizationName,
		runnerPodClient:  runnerPodClient,
	}
}
