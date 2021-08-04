package hooks

import (
	constants "github.com/cybozu-go/meows"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("mutate Pod webhook", func() {
	It("should add token to env", func() {
		By("creating Pod with webhook label")
		pn := "p0"
		ns := "default"

		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pn,
				Namespace: ns,
				Labels: map[string]string{
					constants.RunnerOrgLabelKey:  organizationName,
					constants.RunnerRepoLabelKey: "repo",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  constants.RunnerContainerName,
						Image: "sample:latest",
					},
					{
						Name:  "should-not-be-added",
						Image: "sample:latest",
					},
				},
			},
		}

		err := k8sClient.Create(ctx, &pod)
		Expect(err).NotTo(HaveOccurred())

		ret := &corev1.Pod{}
		err = k8sClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pn,
				Namespace: ns,
			},
			ret,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(ret.Spec.Containers).To(HaveLen(2))
		c0 := ret.Spec.Containers[0]
		Expect(c0.Env).To(HaveLen(1))
		Expect(c0.Env[0].Name).To(Equal(constants.RunnerTokenEnvName))
		c1 := ret.Spec.Containers[1]
		Expect(c1.Env).To(HaveLen(0))
	})

	It("should not add token to env", func() {
		{
			pn := "p1"
			ns := "default"

			By("creating Pod without repo label")
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pn,
					Namespace: ns,
					Labels: map[string]string{
						constants.RunnerOrgLabelKey: organizationName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "should-not-be-added",
							Image: "sample:latest",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, &pod)
			Expect(err).NotTo(HaveOccurred())

			ret := &corev1.Pod{}
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{
					Name:      pn,
					Namespace: ns,
				},
				ret,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret.Spec.Containers).To(HaveLen(1))
			c0 := ret.Spec.Containers[0]
			Expect(c0.Env).To(HaveLen(0))

		}
		{
			pn := "p2"
			ns := "default"

			By("creating Pod without org label")
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pn,
					Namespace: ns,
					Labels: map[string]string{
						constants.RunnerRepoLabelKey: "repo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "should-not-be-added",
							Image: "sample:latest",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, &pod)
			Expect(err).NotTo(HaveOccurred())

			ret := &corev1.Pod{}
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{
					Name:      pn,
					Namespace: ns,
				},
				ret,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret.Spec.Containers).To(HaveLen(1))
			c0 := ret.Spec.Containers[0]
			Expect(c0.Env).To(HaveLen(0))
		}
		{
			pn := "p3"
			ns := "default"

			By("creating Pod with non-target org label")
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pn,
					Namespace: ns,
					Labels: map[string]string{
						constants.RunnerOrgLabelKey:  "incorrect-fake-org",
						constants.RunnerRepoLabelKey: "repo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "should-not-be-added",
							Image: "sample:latest",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, &pod)
			Expect(err).NotTo(HaveOccurred())

			ret := &corev1.Pod{}
			err = k8sClient.Get(
				ctx,
				types.NamespacedName{
					Name:      pn,
					Namespace: ns,
				},
				ret,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(ret.Spec.Containers).To(HaveLen(1))
			c0 := ret.Spec.Containers[0]
			Expect(c0.Env).To(HaveLen(0))
		}
	})
})
