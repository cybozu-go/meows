package controllers

import (
	"context"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
	gogithub "github.com/google/go-github/v33/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RunnerSweeper runner", func() {
	organizationName := "runnersweep-org"
	repositoryName := "runnersweep-repo"
	githubClient := github.NewFakeClient(organizationName)

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

		sweeper := NewRunnerSweeper(
			mgr.GetClient(),
			ctrl.Log.WithName("runner-sweeper"),
			time.Second,
			githubClient,
			[]string{repositoryName},
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

	It("should delete unused runner", func() {
		By("creating namespaces")
		createNamespaces(ctx, "ns0", "ns1")

		testCases := []struct {
			name            string
			podsBefore      []types.NamespacedName
			runnersBefore   []*gogithub.Runner
			runnersExpected map[string]struct{}
		}{
			{
				"confirming runners are deleted",
				[]types.NamespacedName{
					{Namespace: "ns0", Name: "p00"},
					{Namespace: "ns0", Name: "p01"},
					{Namespace: "ns1", Name: "p10"},
					{Namespace: "ns1", Name: "p11"},
				},
				[]*gogithub.Runner{
					{Name: strPtr("p00"), ID: int64Ptr(1), Status: strPtr(statusOnline)},
					{Name: strPtr("p10"), ID: int64Ptr(2), Status: strPtr(statusOffline)},
					{Name: strPtr("oldonline0"), ID: int64Ptr(3), Status: strPtr(statusOnline)},
					{Name: strPtr("oldoffline0"), ID: int64Ptr(4), Status: strPtr(statusOffline)},
				},
				map[string]struct{}{
					"p00":        {},
					"p10":        {},
					"oldonline0": {},
				},
			},
			{
				"confirming runners are deleted multiple times",
				[]types.NamespacedName{
					{Namespace: "ns0", Name: "p00"},
					{Namespace: "ns0", Name: "p01"},
					{Namespace: "ns1", Name: "p10"},
					{Namespace: "ns1", Name: "p11"},
				},
				[]*gogithub.Runner{
					{Name: strPtr("p00"), ID: int64Ptr(1), Status: strPtr(statusOnline)},
					{Name: strPtr("p10"), ID: int64Ptr(2), Status: strPtr(statusOffline)},
					{Name: strPtr("oldonline0"), ID: int64Ptr(3), Status: strPtr(statusOnline)},
					{Name: strPtr("oldoffline0"), ID: int64Ptr(4), Status: strPtr(statusOffline)},
				},
				map[string]struct{}{
					"p00":        {},
					"p10":        {},
					"oldonline0": {},
				},
			},
		}

		for _, tt := range testCases {
			By(tt.name)
			createPods(ctx, tt.podsBefore, organizationName, repositoryName)
			githubClient.SetRunners(map[string][]*gogithub.Runner{repositoryName: tt.runnersBefore})

			Eventually(func() error {
				runnersActual, _ := githubClient.ListRunners(ctx, repositoryName)
				if len(tt.runnersExpected) != len(runnersActual) {
					return fmt.Errorf("length mismatch: expected %#v, actual %#v", tt.runnersExpected, makeNameList(runnersActual))
				}
				for _, runner := range runnersActual {
					if _, ok := tt.runnersExpected[*runner.Name]; !ok {
						return fmt.Errorf("%s should not exist", *runner.Name)
					}
				}
				return nil
			}).ShouldNot(HaveOccurred())

			deletePods(ctx, tt.podsBefore)

			// sleep until one loop certainly finishes
			time.Sleep(2 * time.Second)
		}
	})
})

func deletePods(ctx context.Context, namespacedNames []types.NamespacedName) {
	for _, n := range namespacedNames {
		pod := &corev1.Pod{}
		pod.Name = n.Name
		pod.Namespace = n.Namespace
		ExpectWithOffset(1, k8sClient.Delete(ctx, pod)).To(Succeed())
		EventuallyWithOffset(1, func() bool {
			err := k8sClient.Get(ctx, n, &corev1.Pod{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	}
}

func createPods(
	ctx context.Context,
	namespacedNames []types.NamespacedName,
	organizationName string,
	repositoryName string,
) {
	for _, n := range namespacedNames {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      n.Name,
				Namespace: n.Namespace,
				Labels: map[string]string{
					constants.RunnerOrgLabelKey:  organizationName,
					constants.RunnerRepoLabelKey: repositoryName,
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
		err := k8sClient.Create(ctx, pod)
		ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func makeNameList(runners []*gogithub.Runner) []string {
	l := make([]string, len(runners))
	for i, v := range runners {
		l[i] = *v.Name
	}
	return l
}
