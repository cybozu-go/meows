package controllers

import (
	"context"
	"time"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RunnerPool reconciler", func() {
	ctx := context.Background()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		rpr := RunnerPoolReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("RunnerPool"),
			Scheme: mgr.GetScheme(),
		}
		err = rpr.SetupWithManager(mgr)
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

	It("should create Deployment", func() {
		name := "runnerpool-0"
		{
			By("deploying RunnerPool resource")
			rp := &actionsv1alpha1.RunnerPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: actionsv1alpha1.RunnerPoolSpec{
					DeploymentSpec: actionsv1alpha1.DeploymentSpec{
						Replicas: int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": name,
							},
						},
						Template: actionsv1alpha1.PodTemplateSpec{
							ObjectMeta: actionsv1alpha1.ObjectMeta{
								Labels: map[string]string{
									"app": name,
								},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  controllerContainerName,
										Image: "sample:latest",
									},
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, rp)
			Expect(err).To(Succeed())
		}

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		nsn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		Eventually(func() error {
			rp := new(actionsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, nsn, rp); err != nil {
				return err
			}

			return k8sClient.Get(ctx, nsn, d)
		}).Should(Succeed())

		Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(1))
		v := d.Spec.Template.Spec.Volumes[0]
		Expect(v.Name).To(Equal(actionsTokenVolumeName))
		Expect(v.VolumeSource.Secret.SecretName).To(Equal(actionsTokenSecretName))

		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))
		c := d.Spec.Template.Spec.Containers[0]
		Expect(c.VolumeMounts).To(HaveLen(1))
		Expect(c.VolumeMounts[0].Name).To(Equal(actionsTokenVolumeName))
		Expect(c.VolumeMounts[0].MountPath).To(Equal(actionsTokenMountPath))
	})
})
