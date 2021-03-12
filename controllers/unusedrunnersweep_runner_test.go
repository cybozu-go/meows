package controllers

import (
	"context"
	"time"

	"github.com/cybozu-go/github-actions-controller/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("UnusedRunnerSweeper runner", func() {
	const organizationName = "org"
	ctx := context.Background()

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
			github.NewFakeClient(),
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

	It("should delete token multiple times ", func() {
	})
})
