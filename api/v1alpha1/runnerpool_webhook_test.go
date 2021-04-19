package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("validate RunnerPool webhook with ", func() {
	It("should deny runnerpool with invalid repository name", func() {
		name := "runnerpool-0"
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: RunnerPoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				RepositoryName: "invalid-repository",
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
		Expect(err).To(HaveOccurred())
	})

	It("should deny runnerpool with invalid container name", func() {
		name := "runnerpool-1"
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
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
		name := "runnerpool-2"
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
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
		name := "runnerpool-3"
		rp := RunnerPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
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
})
