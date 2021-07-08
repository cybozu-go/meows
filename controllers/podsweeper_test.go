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
	var runnerPodClient *RunnerPodClientMock
	var mgrCtx context.Context
	var mgrCancel context.CancelFunc

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		runnerPodClient = NewRunnerPodClientMock(time.Time{})

		sweeper := &PodSweeper{
			k8sClient:        mgr.GetClient(),
			log:              ctrl.Log.WithName("pod-sweeper"),
			interval:         time.Second,
			organizationName: organizationName,
			runnerPodClient:  runnerPodClient,
		}
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
			deletionTime time.Time
		}{
			{
				"API return time aleready past",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample1",
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
				time.Now().UTC(),
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
			deletionTime time.Time
		}{
			{
				"without labels",
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sample2",
						Namespace: namespace,
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
				time.Time{},
			},
			{
				"API return time that is zero",
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
				time.Time{},
			},
			{
				"API return future time",
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
				time.Now().Add(24 * time.Hour).UTC(),
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
	deletionTime time.Time
}

func (c *RunnerPodClientMock) GetDeletionTime(ctx context.Context, ip string) (time.Time, error) {
	return c.deletionTime, nil
}

func (c *RunnerPodClientMock) PutDeletionTime(ctx context.Context, ip string, tm time.Time) error {
	c.deletionTime = tm
	return nil
}

func NewRunnerPodClientMock(deletionTime time.Time) *RunnerPodClientMock {
	return &RunnerPodClientMock{
		deletionTime: deletionTime,
	}
}
