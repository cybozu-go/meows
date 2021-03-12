package hooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
)

var _ = Describe("mutate Pod webhook", func() {
	It("should add token to env", func() {
		By("creating Pod with webhook label")
		pn := "p0"
		ns := "default"
		cn := "should-not-be-added"

		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pn,
				Namespace: ns,
				Labels: map[string]string{
					actionscontroller.RunnerRepoLabelKey: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  actionscontroller.RunnerContainerName,
						Image: "sample:latest",
					},
					{
						Name:  cn,
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
		Expect(c0.Env[0].Name).To(Equal(actionscontroller.RunnerTokenEnvName))
		c1 := ret.Spec.Containers[1]
		Expect(c1.Env).To(HaveLen(0))
	})

	It("should not add token to env", func() {
		By("creating Pod without webhook label")
		pn := "p1"
		ns := "default"
		cn := "should-not-be-added"

		By("creating Pod")
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pn,
				Namespace: ns,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  cn,
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
	})
})
