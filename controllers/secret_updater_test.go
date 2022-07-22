package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("SecretUpdater", func() {
	ctx := context.Background()

	It("should create namespace", func() {
		createNamespaces(ctx, "secretupdater-test")
	})

	It("should update secret", func() {
		testCase := []struct {
			name        string
			annotations map[string]string
		}{
			{
				name:        "empty",
				annotations: nil,
			},
			{
				name: "invalid",
				annotations: map[string]string{
					"meows.cybozu.com/expires-at": "invalid",
				},
			},
		}

		githubClientFactory := github.NewFakeClientFactory()
		githubClientFactory.SetExpiredAtDuration(100 * time.Hour)
		secretUpdater := NewSecretUpdater(ctrl.Log, k8sClient, githubClientFactory)

		for _, tc := range testCase {
			rp := makeRunnerPoolWithOrganization(tc.name, "secretupdater-test", "test-org")

			By("creating secret")
			beforeSec := new(corev1.Secret)
			beforeSec.SetName(rp.GetRunnerSecretName())
			beforeSec.SetNamespace("secretupdater-test")
			beforeSec.SetAnnotations(tc.annotations)
			Expect(k8sClient.Create(ctx, beforeSec)).To(Succeed(), tc.name)

			By("starting secret updater")
			secretUpdater.Start(rp, nil)
			time.Sleep(3 * time.Second)

			By("getting secret")
			afterSec := new(corev1.Secret)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rp.GetRunnerSecretName(), Namespace: "secretupdater-test"}, afterSec)).To(Succeed(), tc.name)

			By("checking the secret was updated")
			tmStr := afterSec.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
			tm, err := time.Parse(time.RFC3339, tmStr)
			Expect(err).NotTo(HaveOccurred(), tc.name)
			expectedExpiresAt := time.Now().Add(100 * time.Hour)
			Expect(tm).To(BeTemporally("~", expectedExpiresAt, 20*time.Second), tc.name)

			By("stopping secret updater")
			secretUpdater.Stop(rp)
		}
	})

	It("should or not update secret", func() {
		testCase := []struct {
			name              string
			shouldUpdate      bool
			expiresAtDuration time.Duration
		}{
			{
				name:              "time1",
				shouldUpdate:      true,
				expiresAtDuration: -10 * time.Minute,
			},
			{
				name:              "time2",
				shouldUpdate:      true,
				expiresAtDuration: 0 * time.Minute,
			},
			{
				name:              "time3",
				shouldUpdate:      true,
				expiresAtDuration: 5 * time.Minute,
			},
			{
				name:              "time4",
				shouldUpdate:      false,
				expiresAtDuration: 20 * time.Minute,
			},
			{
				name:              "time5",
				shouldUpdate:      false,
				expiresAtDuration: 1 * time.Hour,
			},
		}

		githubClientFactory := github.NewFakeClientFactory()
		githubClientFactory.SetExpiredAtDuration(100 * time.Hour)
		secretUpdater := NewSecretUpdater(ctrl.Log, k8sClient, githubClientFactory)

		for _, tc := range testCase {
			rp := makeRunnerPoolWithRepository(tc.name, "secretupdater-test", "owner/test-repo")
			rp.Spec.Organization = "test-org"
			baseTime := time.Now().Add(tc.expiresAtDuration).Format(time.RFC3339)

			By("creating secret")
			beforeSec := new(corev1.Secret)
			beforeSec.SetName(rp.GetRunnerSecretName())
			beforeSec.SetNamespace("secretupdater-test")
			beforeSec.SetAnnotations(map[string]string{"meows.cybozu.com/expires-at": baseTime})
			Expect(k8sClient.Create(ctx, beforeSec)).To(Succeed(), tc.name)

			By("starting secret updater")
			secretUpdater.Start(rp, nil)
			time.Sleep(3 * time.Second)

			By("getting secret")
			afterSec := new(corev1.Secret)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: rp.GetRunnerSecretName(), Namespace: "secretupdater-test"}, afterSec)).To(Succeed(), tc.name)

			if tc.shouldUpdate {
				By("checking the secret was updated")
				tmStr := afterSec.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
				Expect(tmStr).NotTo(Equal(baseTime), tc.name)
				tm, err := time.Parse(time.RFC3339, tmStr)
				Expect(err).NotTo(HaveOccurred(), tc.name)
				expectedExpiresAt := time.Now().Add(100 * time.Hour)
				Expect(tm).To(BeTemporally("~", expectedExpiresAt, 20*time.Second), tc.name)
			} else {
				By("checking the secret was not updated")
				tmStr := afterSec.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
				Expect(tmStr).To(Equal(baseTime), tc.name)
			}

			By("stopping secret updater")
			secretUpdater.Stop(rp)
		}
	})
})
