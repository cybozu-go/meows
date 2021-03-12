package controllers

import (
	"context"
	"fmt"
	"time"

	gogithub "github.com/google/go-github/v33/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
)

var _ = Describe("UnusedRunnerSweeper runner", func() {
	ctx := context.Background()
	organizationName := "org"
	repositoryName := "repo"

	githubClient := github.NewFakeClient()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		sweeper := NewUnusedRunnerSweeper(
			mgr.GetClient(),
			ctrl.Log.WithName("actions-token-updator"),
			time.Second,
			githubClient,
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
		time.Sleep(100 * time.Millisecond)
	})

	It("should delete unused token", func() {
		testCases := []struct {
			podsBefore      []types.NamespacedName
			runnersBefore   []*gogithub.Runner
			runnersExpected map[string]struct{}
		}{
			{
				[]types.NamespacedName{
					{Namespace: "ns0", Name: "p00"},
					{Namespace: "ns0", Name: "p01"},
					{Namespace: "ns1", Name: "p10"},
					{Namespace: "ns1", Name: "p11"},
				},
				[]*gogithub.Runner{
					{Name: strPtr("p00"), ID: int64Ptr(1)},
					{Name: strPtr("p10"), ID: int64Ptr(2)},
					{Name: strPtr("old0"), ID: int64Ptr(3)},
					{Name: strPtr("old1"), ID: int64Ptr(4)},
				},
				map[string]struct{}{
					"p00": {},
					"p10": {},
				},
			},
		}

		for _, tt := range testCases {
			createNamespaces(ctx, tt.podsBefore)
			createPods(ctx, tt.podsBefore, organizationName, repositoryName)
			githubClient.SetRunners(map[string][]*gogithub.Runner{repositoryName: tt.runnersBefore})

			Eventually(func() error {
				runnersActual, _ := githubClient.ListRunners(ctx, repositoryName)
				if len(tt.runnersExpected) != len(runnersActual) {
					return fmt.Errorf("length mismatch: expected %#v, actual %#v", tt.runnersExpected, runnersActual)
				}
				for _, runner := range runnersActual {
					if _, ok := tt.runnersExpected[*runner.Name]; !ok {
						return fmt.Errorf("%s should not exist", *runner.Name)
					}
				}
				return nil
			}).ShouldNot(HaveOccurred())
		}
	})
})

func createNamespaces(ctx context.Context, namespacedNames []types.NamespacedName) {
	namespaces := make(map[string]struct{})
	for _, n := range namespacedNames {
		namespaces[n.Namespace] = struct{}{}
	}
	for n := range namespaces {
		ns := &corev1.Namespace{}
		ns.Name = n
		err := k8sClient.Create(ctx, ns)
		ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
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
					actionscontroller.RunnerOrgLabelKey:  organizationName,
					actionscontroller.RunnerRepoLabelKey: repositoryName,
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
