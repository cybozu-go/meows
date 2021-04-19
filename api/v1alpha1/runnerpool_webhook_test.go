package v1alpha1

import (
	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("validate RunnerPool webhook with ", func() {
	name := "runnerpool-test"
	namespace := "default"
	It("should deny runnerpool with invalid container name", func() {
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: RunnerPoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				RepositoryName: "test-repository2",
				Template: PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "sample",
								Image: namespace,
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, &rp)
		Expect(err).To(HaveOccurred())
	})

	It("should deny runnerpool with reserved env name", func() {
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: RunnerPoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				RepositoryName: "test-repository2",
				Template: PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "sample",
								Image: "sample:latest",
								Env: []corev1.EnvVar{
									{
										Name:  "POD_NAME",
										Value: "pod_name",
									},
								},
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, &rp)
		Expect(err).To(HaveOccurred())
	})

	It("should accept runnerpool", func() {
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: RunnerPoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				RepositoryName: "test-repository2",
				Template: PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "runner",
								Image: "sample:latest",
							},
						},
					},
				},
			},
		}
		err := k8sClient.Create(ctx, &rp)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should confirm runnerpool finalizer", func() {
		nsn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		rp := &RunnerPool{}
		err := k8sClient.Get(ctx, nsn, rp)
		Expect(err).NotTo(HaveOccurred())
		Expect(1).To(Equal(len(rp.ObjectMeta.Finalizers)))
		Expect(rp.ObjectMeta.Finalizers[0]).To(Equal(constants.RunnerPoolFinalizer))
	})
})
