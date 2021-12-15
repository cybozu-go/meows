package controllers

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"time"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type runnerManagerMock struct {
	started     map[string]bool
	githubCreds map[string]*github.ClientCredential
}

func newRunnerManagerMock() *runnerManagerMock {
	return &runnerManagerMock{
		started:     map[string]bool{},
		githubCreds: map[string]*github.ClientCredential{},
	}
}

func (m *runnerManagerMock) StartOrUpdate(rp *meowsv1alpha1.RunnerPool, cred *github.ClientCredential) error {
	rpNamespacedName := rp.Namespace + "/" + rp.Name
	m.started[rpNamespacedName] = true
	m.githubCreds[rpNamespacedName] = cred
	return nil
}

func (m *runnerManagerMock) Stop(rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedName := rp.Namespace + "/" + rp.Name
	delete(m.started, rpNamespacedName)
	delete(m.githubCreds, rpNamespacedName)
	return nil
}

func (m *runnerManagerMock) StopAll() {
}

type secretUpdaterMock struct {
	k8sClient   client.Client
	started     map[string]bool
	githubCreds map[string]*github.ClientCredential
}

func newSecretUpdaterMock(c client.Client) *secretUpdaterMock {
	return &secretUpdaterMock{
		k8sClient:   c,
		started:     map[string]bool{},
		githubCreds: map[string]*github.ClientCredential{},
	}
}

func (m *secretUpdaterMock) Start(rp *meowsv1alpha1.RunnerPool, cred *github.ClientCredential) error {
	rpNamespacedName := rp.Namespace + "/" + rp.Name
	m.started[rpNamespacedName] = true
	m.githubCreds[rpNamespacedName] = cred

	ctx := context.Background()
	s := new(corev1.Secret)
	err := m.k8sClient.Get(ctx, types.NamespacedName{Namespace: rp.Namespace, Name: rp.GetRunnerSecretName()}, s)
	if err != nil {
		return err
	}
	newS := s.DeepCopy()
	newS.Annotations = mergeMap(s.Annotations, map[string]string{
		constants.RunnerSecretExpiresAtAnnotationKey: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})
	newS.StringData = map[string]string{
		constants.RunnerTokenFileName: "dummy token",
	}
	patch := client.MergeFrom(s)
	return m.k8sClient.Patch(ctx, newS, patch)
}

func (m *secretUpdaterMock) Stop(rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedName := rp.Namespace + "/" + rp.Name
	delete(m.started, rpNamespacedName)
	delete(m.githubCreds, rpNamespacedName)
	return nil
}

func (m *secretUpdaterMock) StopAll() {
}

var _ = Describe("RunnerPool reconciler", func() {
	namespace := "runnerpool-ns"
	runnerPoolName := "runnerpool-1"
	secretName := "runner-token-" + runnerPoolName
	deploymentName := "runnerpool-1"
	defaultRunnerImage := "sample:latest"
	serviceAccountName := "customized-sa"
	wait := 10 * time.Second
	var mockManager *runnerManagerMock
	var mockUpdater *secretUpdaterMock

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

		mockManager = newRunnerManagerMock()
		mockUpdater = newSecretUpdaterMock(mgr.GetClient())

		r := NewRunnerPoolReconciler(
			ctrl.Log,
			mgr.GetClient(),
			mgr.GetScheme(),
			defaultRunnerImage,
			RunnerManager(mockManager),
			SecretUpdater(mockUpdater),
			regexp.MustCompile(`^test-org$`),
			regexp.MustCompile(`^test-org/.*`),
		)
		Expect(r.SetupWithManager(mgr)).To(Succeed())

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

		By("creating default credential secret")
		appSecret := new(corev1.Secret)
		appSecret.SetName("meows-github-cred") // Default name
		appSecret.SetNamespace(namespace)
		appSecret.StringData = map[string]string{
			"app-id":              "1234",
			"app-installation-id": "5678",
			"app-private-key":     "dummy-private-key",
		}
		Expect(k8sClient.Create(ctx, appSecret)).To(Succeed())

		By("creating another credential secret")
		patSecret := new(corev1.Secret)
		patSecret.SetName("github-cred-foo")
		patSecret.SetNamespace(namespace)
		patSecret.StringData = map[string]string{
			"token": "dummy-pat",
		}
		Expect(k8sClient.Create(ctx, patSecret)).To(Succeed())
	})

	It("should create Deployment from minimal RunnerPool", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPool(runnerPoolName, namespace)
		rp.Spec.Repository = "test-org/test-repo"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("waiting the RunnerPool become Bound")
		Eventually(func() error {
			rp := new(meowsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if !rp.Status.Bound {
				return errors.New(`status "bound" should be true`)
			}
			return nil
		}).Should(Succeed())
		time.Sleep(wait) // Wait for the reconciliation to run a few times. Please check the controller's log.

		By("getting the created Secret")
		s := new(corev1.Secret)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, s)).To(Succeed())

		By("checking that a runner pool is the owner of a secret")
		Expect(s.OwnerReferences).To(HaveLen(1))
		Expect(s.OwnerReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Kind": Equal("RunnerPool"),
			"Name": Equal(runnerPoolName),
		}))

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, d)).To(Succeed())

		By("confirming the Deployment's manifests")
		// labels
		Expect(d.Labels).To(MatchAllKeys(Keys{
			constants.AppNameLabelKey:      Equal(constants.AppName),
			constants.AppComponentLabelKey: Equal(constants.AppComponentRunner),
			constants.AppInstanceLabelKey:  Equal(runnerPoolName),
		}))
		Expect(d.Spec.Selector.MatchLabels).To(MatchAllKeys(Keys{
			constants.AppNameLabelKey:      Equal(constants.AppName),
			constants.AppComponentLabelKey: Equal(constants.AppComponentRunner),
			constants.AppInstanceLabelKey:  Equal(runnerPoolName),
		}))
		Expect(d.Spec.Template.Labels).To(MatchAllKeys(Keys{
			constants.AppNameLabelKey:      Equal(constants.AppName),
			constants.AppComponentLabelKey: Equal(constants.AppComponentRunner),
			constants.AppInstanceLabelKey:  Equal(runnerPoolName),
		}))

		// deployment/pod spec
		Expect(d.Spec.Replicas).To(PointTo(BeNumerically("==", 1)))
		Expect(d.Spec.Template.Spec).To(MatchFields(IgnoreExtras, Fields{
			"ServiceAccountName":           Equal("default"),
			"AutomountServiceAccountToken": BeNil(),
			"ImagePullSecrets":             BeEmpty(),
			"Volumes": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("var-dir"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"EmptyDir": Not(BeNil()),
					}),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("work-dir"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"EmptyDir": Not(BeNil()),
					}),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(secretName),
				}),
			}),
		}))

		// runner container spec
		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(d.Spec.Template.Spec.Containers[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name":            Equal(constants.RunnerContainerName),
			"Image":           Equal(defaultRunnerImage),
			"ImagePullPolicy": Equal(corev1.PullAlways),
			"SecurityContext": BeNil(),
			"Env": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(constants.PodNameEnvName),
					"ValueFrom": PointTo(MatchFields(IgnoreExtras, Fields{
						"FieldRef": PointTo(MatchFields(IgnoreExtras, Fields{
							"FieldPath": Equal("metadata.name"),
						})),
					})),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(constants.PodNamespaceEnvName),
					"ValueFrom": PointTo(MatchFields(IgnoreExtras, Fields{
						"FieldRef": PointTo(MatchFields(IgnoreExtras, Fields{
							"FieldPath": Equal("metadata.namespace"),
						})),
					})),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerPoolNameEnvName),
					"Value": Equal(runnerPoolName),
				}),
				"3": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerOptionEnvName),
					"Value": Equal("{}"),
				}),
				"4": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerRepoEnvName),
					"Value": Equal("test-org/test-repo"),
				}),
			}),
			"Resources": MatchAllFields(Fields{
				"Limits":   BeEmpty(),
				"Requests": BeEmpty(),
			}),
			"Ports": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Protocol":      Equal(corev1.ProtocolTCP),
					"Name":          Equal(constants.RunnerMetricsPortName),
					"ContainerPort": BeNumerically("==", constants.RunnerListenPort),
				}),
			}),
			"VolumeMounts": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("var-dir"),
					"MountPath": Equal(constants.RunnerVarDirPath),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("work-dir"),
					"MountPath": Equal("/runner/_work"),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal(secretName),
					"MountPath": Equal(filepath.Join(constants.RunnerVarDirPath, constants.SecretsDirName)),
				}),
			}),
		}))

		By("checking that a runner pool is the owner of a deployment")
		Expect(d.OwnerReferences).To(HaveLen(1))
		Expect(d.OwnerReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Kind": Equal("RunnerPool"),
			"Name": Equal(runnerPoolName),
		}))

		By("checking sub-processes are started")
		Expect(mockManager.started).To(HaveKey(namespace + "/" + runnerPoolName))
		Expect(mockUpdater.started).To(HaveKey(namespace + "/" + runnerPoolName))

		By("checking credentials have been passed to sub-processes")
		Expect(mockManager.githubCreds[namespace+"/"+runnerPoolName]).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"AppID":             Equal(int64(1234)),
			"AppInstallationID": Equal(int64(5678)),
			"PrivateKey":        Equal([]byte("dummy-private-key")),
		})))
		Expect(mockUpdater.githubCreds[namespace+"/"+runnerPoolName]).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"AppID":             Equal(int64(1234)),
			"AppInstallationID": Equal(int64(5678)),
			"PrivateKey":        Equal([]byte("dummy-private-key")),
		})))

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)

		By("checking sub-processes are stopped")
		Expect(mockManager.started).NotTo(HaveKey(namespace + "/" + runnerPoolName))
		Expect(mockUpdater.started).NotTo(HaveKey(namespace + "/" + runnerPoolName))
	})

	It("should create Deployment from maximum RunnerPool", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPool(runnerPoolName, namespace)
		rp.Spec.Organization = "test-org"
		rp.Spec.CredentialSecretName = "github-cred-foo"
		rp.Spec.Replicas = 3
		rp.Spec.SetupCommand = []string{"command", "arg1", "args2"}
		rp.Spec.SlackNotification.Enable = true
		rp.Spec.SlackNotification.Channel = "#test"
		rp.Spec.SlackNotification.ExtendDuration = "20m"
		rp.Spec.Template.ObjectMeta.Labels = map[string]string{
			"test-label": "test",
		}
		rp.Spec.Template.ObjectMeta.Annotations = map[string]string{
			"test-annotation": "test",
		}
		rp.Spec.Template.Image = "sample:devel"
		rp.Spec.Template.ImagePullPolicy = corev1.PullIfNotPresent
		rp.Spec.Template.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: "image-pull-secret1"},
		}
		rp.Spec.Template.SecurityContext = &corev1.SecurityContext{
			Privileged: pointer.BoolPtr(true),
		}
		rp.Spec.Template.Env = []corev1.EnvVar{
			{Name: "ENV_VAR", Value: "value"},
		}
		rp.Spec.Template.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("100"),
			},
		}
		rp.Spec.Template.VolumeMounts = []corev1.VolumeMount{
			{Name: "volume1", MountPath: "/volume1"},
			{Name: "volume2", MountPath: "/volume2"},
		}
		rp.Spec.Template.Volumes = []corev1.Volume{
			{Name: "volume1", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			{Name: "volume2", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
		rp.Spec.Template.WorkVolume = &corev1.VolumeSource{
			Ephemeral: &corev1.EphemeralVolumeSource{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vol",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("100Mi"),
							},
						},
					},
				},
			},
		}
		rp.Spec.Template.ServiceAccountName = serviceAccountName
		rp.Spec.Template.AutomountServiceAccountToken = pointer.BoolPtr(false)
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("waiting the RunnerPool become Bound")
		Eventually(func() error {
			rp := new(meowsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if !rp.Status.Bound {
				return errors.New(`status "bound" should be true`)
			}
			return nil
		}).Should(Succeed())
		time.Sleep(wait) // Wait for the reconciliation to run a few times. Please check the controller's log.

		By("getting the created Secret")
		s := new(corev1.Secret)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, s)).To(Succeed())

		By("checking that a runner pool is the owner of a secret")
		Expect(s.OwnerReferences).To(HaveLen(1))
		Expect(s.OwnerReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Kind": Equal("RunnerPool"),
			"Name": Equal(runnerPoolName),
		}))

		By("getting the created Deployment")
		d := new(appsv1.Deployment)
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, d)).To(Succeed())

		By("confirming the Deployment's manifests")
		// labels (omit)

		// deployment/pod spec
		Expect(d.Spec.Replicas).To(PointTo(BeNumerically("==", 3)))
		Expect(d.Spec.Template.Spec).To(MatchFields(IgnoreExtras, Fields{
			"ServiceAccountName":           Equal(serviceAccountName),
			"AutomountServiceAccountToken": PointTo(BeFalse()),
			"ImagePullSecrets": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("image-pull-secret1"),
				}),
			}),
			"Volumes": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("volume1"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"EmptyDir": Not(BeNil()),
					}),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("volume2"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"EmptyDir": Not(BeNil()),
					}),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("var-dir"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"EmptyDir": Not(BeNil()),
					}),
				}),
				"3": MatchFields(IgnoreExtras, Fields{
					"Name": Equal("work-dir"),
					"VolumeSource": MatchFields(IgnoreExtras, Fields{
						"Ephemeral": Not(BeNil()),
					}),
				}),
				"4": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(secretName),
				}),
			}),
		}))

		// runner container spec
		Expect(d.Spec.Template).To(MatchFields(IgnoreExtras, Fields{
			"ObjectMeta": MatchFields(IgnoreExtras, Fields{
				"Labels": MatchAllKeys(Keys{
					constants.AppNameLabelKey:      Equal(constants.AppName),
					constants.AppComponentLabelKey: Equal(constants.AppComponentRunner),
					constants.AppInstanceLabelKey:  Equal(rp.Name),
					"test-label":                   Equal("test"),
				}),
				"Annotations": MatchAllKeys(Keys{
					"test-annotation": Equal("test"),
				}),
			}),
		}))
		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(d.Spec.Template.Spec.Containers[0]).To(MatchFields(IgnoreExtras, Fields{
			"Name":            Equal(constants.RunnerContainerName),
			"Image":           Equal("sample:devel"),
			"ImagePullPolicy": Equal(corev1.PullIfNotPresent),
			"SecurityContext": PointTo(MatchFields(IgnoreExtras, Fields{
				"Privileged": PointTo(BeTrue()),
			})),
			"Env": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(constants.PodNameEnvName),
					"ValueFrom": PointTo(MatchFields(IgnoreExtras, Fields{
						"FieldRef": PointTo(MatchFields(IgnoreExtras, Fields{
							"FieldPath": Equal("metadata.name"),
						})),
					})),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name": Equal(constants.PodNamespaceEnvName),
					"ValueFrom": PointTo(MatchFields(IgnoreExtras, Fields{
						"FieldRef": PointTo(MatchFields(IgnoreExtras, Fields{
							"FieldPath": Equal("metadata.namespace"),
						})),
					})),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerPoolNameEnvName),
					"Value": Equal(runnerPoolName),
				}),
				"3": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerOptionEnvName),
					"Value": Equal("{\"setup_command\":[\"command\",\"arg1\",\"args2\"]}"),
				}),
				"4": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(constants.RunnerOrgEnvName),
					"Value": Equal("test-org"),
				}),
				"5": MatchFields(IgnoreExtras, Fields{
					"Name":  Equal("ENV_VAR"),
					"Value": Equal("value"),
				}),
			}),
			"Resources": MatchAllFields(Fields{
				"Limits": BeEmpty(),
				"Requests": MatchAllKeys(Keys{
					corev1.ResourceMemory: Equal(resource.MustParse("100")),
				}),
			}),
			"Ports": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Protocol":      Equal(corev1.ProtocolTCP),
					"Name":          Equal(constants.RunnerMetricsPortName),
					"ContainerPort": BeNumerically("==", constants.RunnerListenPort),
				}),
			}),
			"VolumeMounts": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("volume1"),
					"MountPath": Equal("/volume1"),
				}),
				"1": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("volume2"),
					"MountPath": Equal("/volume2"),
				}),
				"2": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("var-dir"),
					"MountPath": Equal(constants.RunnerVarDirPath),
				}),
				"3": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal("work-dir"),
					"MountPath": Equal("/runner/_work"),
				}),
				"4": MatchFields(IgnoreExtras, Fields{
					"Name":      Equal(secretName),
					"MountPath": Equal(filepath.Join(constants.RunnerVarDirPath, constants.SecretsDirName)),
				}),
			}),
		}))

		By("checking that a runner pool is the owner of a deployment")
		Expect(d.OwnerReferences).To(HaveLen(1))
		Expect(d.OwnerReferences[0]).To(MatchFields(IgnoreExtras, Fields{
			"Kind": Equal("RunnerPool"),
			"Name": Equal(runnerPoolName),
		}))

		By("checking sub-processes are started")
		Expect(mockManager.started).To(HaveKey(namespace + "/" + runnerPoolName))
		Expect(mockUpdater.started).To(HaveKey(namespace + "/" + runnerPoolName))

		By("checking credentials have been passed to sub-processes")
		Expect(mockManager.githubCreds[namespace+"/"+runnerPoolName]).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"PersonalAccessToken": Equal("dummy-pat"),
		})))
		Expect(mockUpdater.githubCreds[namespace+"/"+runnerPoolName]).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"PersonalAccessToken": Equal("dummy-pat"),
		})))

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)

		By("checking sub-processes are stopped")
		Expect(mockManager.started).NotTo(HaveKey(namespace + "/" + runnerPoolName))
		Expect(mockUpdater.started).NotTo(HaveKey(namespace + "/" + runnerPoolName))
	})

	It("should not create Deployment from unpermitted repository", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPool(runnerPoolName, namespace)
		rp.Spec.Repository = "test-org2/test-repo"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("waiting the RunnerPool become Bound")
		Consistently(func() error {
			rp := new(meowsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if rp.Status.Bound {
				return errors.New(`status "bound" should not be true`)
			}
			return nil
		}).Should(Succeed())
		time.Sleep(wait) // Wait for the reconciliation to run a few times. Please check the controller's log.

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)
	})

	It("should not create Deployment from unpermitted organization", func() {
		By("deploying RunnerPool resource")
		rp := makeRunnerPool(runnerPoolName, namespace)
		rp.Spec.Organization = "test-org2"
		Expect(k8sClient.Create(ctx, rp)).To(Succeed())

		By("waiting the RunnerPool become Bound")
		Consistently(func() error {
			rp := new(meowsv1alpha1.RunnerPool)
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerPoolName, Namespace: namespace}, rp); err != nil {
				return err
			}
			if rp.Status.Bound {
				return errors.New(`status "bound" should not be true`)
			}
			return nil
		}).Should(Succeed())
		time.Sleep(wait) // Wait for the reconciliation to run a few times. Please check the controller's log.

		By("deleting the created RunnerPool")
		deleteRunnerPool(ctx, runnerPoolName, namespace)
	})
})
