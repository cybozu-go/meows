package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RunnerManager", func() {
	ctx := context.Background()
	metricsPort := ":12345"
	metricsURL := "http://localhost" + metricsPort

	AfterEach(func() {
		k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns1"))
		k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns2"))
		time.Sleep(500 * time.Millisecond)
	})

	It("should create namespace", func() {
		createNamespaces(ctx, "test-ns1", "test-ns2")
	})

	It("should manage pods and runners", func() {
		type inputPod struct {
			spec         *corev1.Pod
			ip           string
			phase        corev1.PodPhase
			state        string
			finishedAt   time.Time
			deletionTime time.Time
		}
		testCases := []struct {
			name             string
			inputRunnerPools []*meowsv1alpha1.RunnerPool
			inputPods        []*inputPod
			inputRunners     map[string][]*github.Runner // key: "<Owner>/<Repository>"
			expectedPods     []string                    // slice of "<Namespace>/<Pod name>"
			expectedRunners  []string                    // slice of "<Owner>/<Repository>/<Runner name>"
		}{
			{
				name: "delete pods",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1"),
					makeRunnerPoolWithRepository("rp2", "test-ns1", "owner/repo2"),
					makeRunnerPoolWithRecreateDeadline("rp3", "test-ns2", "owner/repo2", "5s"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now()}, // state is debugging.
					{spec: makePod("pod2", "test-ns1", "rp2"), ip: "10.0.0.2", state: "stale"},                                                       // state is stale.
					{spec: makePod("pod3", "test-ns2", "rp3"), ip: "10.0.0.3", state: "running"},                                                     // recreate deadline is exceeded and runner is not exist.
					{spec: makePod("pod4", "test-ns2", "rp3"), ip: "10.0.0.4", state: "running"},                                                     // recreate deadline is exceeded and runner is not busy.
				},
				inputRunners: map[string][]*github.Runner{
					"owner/repo2": {
						{Name: "pod4", ID: 4, Online: true, Busy: false, Labels: []string{"test-ns2/rp3"}},
					},
				},
				expectedPods: nil,
				expectedRunners: []string{
					"owner/repo2/pod4",
				},
			},
			{
				name: "should not delete pods",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1"),
					makeRunnerPoolWithRepository("rp2", "test-ns1", "owner/repo2"),
					makeRunnerPoolWithRecreateDeadline("rp3", "test-ns2", "owner/repo2", "5s"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "initializing"},
					{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.2", state: "running"},
					{spec: makePod("pod3", "test-ns1", "rp2"), ip: "10.0.0.3", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now().Add(24 * time.Hour)},
					{spec: makePod("pod4", "test-ns1", "rp3"), ip: "10.0.0.4", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now()},                     // state is debugging but RunnerPool (test-ns1/rp3) is not exists.
					{spec: makePod("pod1", "test-ns2", "rp1"), ip: "10.0.1.1", state: "stale"},                                                                           // state is stale but RunnerPool (test-ns2/rp1) is not exists.
					{spec: makePod("pod2", "test-ns2", "rp3"), ip: "10.0.1.2", state: "running"},                                                                         // recreate deadline is exceeded but runner is busy.
					{spec: makePod("pod3", "test-ns2", "rp3"), ip: "10.0.1.3", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now().Add(24 * time.Hour)}, // recreate deadline is exceeded but state is debugging.
				},
				inputRunners: map[string][]*github.Runner{
					"owner/repo2": {
						{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns2/rp3"}},
					},
				},
				expectedPods: []string{
					"test-ns1/pod1",
					"test-ns1/pod2",
					"test-ns1/pod3",
					"test-ns1/pod4",
					"test-ns2/pod1",
					"test-ns2/pod2",
					"test-ns2/pod3",
				},
				expectedRunners: []string{
					"owner/repo2/pod2",
				},
			},
			{
				name: "delete runners",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1"),
					makeRunnerPoolWithRepository("rp2", "test-ns1", "owner/repo2"),
				},
				inputPods: nil,
				inputRunners: map[string][]*github.Runner{
					"owner/repo1": {
						{Name: "pod1", ID: 1, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod does not exist, offline
						{Name: "pod2", ID: 2, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod does not exist, offline
					},
					"owner/repo2": {
						{Name: "pod3", ID: 3, Online: false, Busy: false, Labels: []string{"test-ns1/rp2"}}, // pod does not exist, offline
					},
				},
				expectedPods:    nil,
				expectedRunners: nil,
			},
			{
				name: "should not delete runners",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1"),
					makeRunnerPoolWithRepository("rp2", "test-ns1", "owner/repo2"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "running"},
					{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.2", state: "running"},
				},
				inputRunners: map[string][]*github.Runner{
					"owner/repo1": {
						{Name: "pod1", ID: 1, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod exists
						{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},   // pod exists
						{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},  // pod does not exist, but online
					},
					"owner/repo2": {
						{Name: "pod1", ID: 4, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}},
						{Name: "pod2", ID: 5, Online: false, Busy: false, Labels: []string{"test-ns1/rp3"}},
						{Name: "pod3", ID: 6, Online: false, Busy: false, Labels: []string{}},
					},
				},
				expectedPods: []string{
					"test-ns1/pod1",
					"test-ns1/pod2",
				},
				expectedRunners: []string{
					"owner/repo1/pod1",
					"owner/repo1/pod2",
					"owner/repo1/pod3",
					"owner/repo2/pod1",
					"owner/repo2/pod2",
					"owner/repo2/pod3",
				},
			},
			{
				name: "should ignore not running pod",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "running", phase: corev1.PodFailed},
					{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.1", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now()},
				},
				inputRunners: nil,
				expectedPods: []string{
					"test-ns1/pod1",
				},
				expectedRunners: nil,
			},
		}

		for _, tt := range testCases {
			By(tt.name)
			ttName := fmt.Sprintf("test case name is '%s'", tt.name)

			By("preparing fake clients")
			runnerPodClient := runner.NewFakeClient()
			githubClientFactory := github.NewFakeClientFactory()
			runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

			By("preparing pods and runners")
			for _, inputPod := range tt.inputPods {
				Expect(k8sClient.Create(ctx, inputPod.spec)).To(Succeed(), ttName)
				created := &corev1.Pod{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: inputPod.spec.Name, Namespace: inputPod.spec.Namespace}, created)).To(Succeed(), ttName)
				created.Status.PodIP = inputPod.ip
				phase := corev1.PodRunning
				if inputPod.phase != "" {
					phase = inputPod.phase
				}
				created.Status.Phase = phase
				Expect(k8sClient.Status().Update(ctx, created)).To(Succeed(), ttName)

				status := runner.Status{
					State: inputPod.state,
				}
				if !inputPod.finishedAt.IsZero() {
					status.FinishedAt = &inputPod.finishedAt
				}
				if !inputPod.deletionTime.IsZero() {
					status.DeletionTime = &inputPod.deletionTime
				}
				runnerPodClient.SetStatus(created.Status.PodIP, &status)
			}
			githubClientFactory.SetRunners(tt.inputRunners)

			By("starting runnerpool manager")
			for _, rp := range tt.inputRunnerPools {
				runnerManager.StartOrUpdate(rp, nil)
			}
			time.Sleep(10 * time.Second) // Wait for the deadline to recreate the pod.

			By("checking pods")
			var actualPodNames []string
			podList := new(corev1.PodList)
			Expect(k8sClient.List(ctx, podList)).To(Succeed(), ttName)
			for i := range podList.Items {
				po := &podList.Items[i]
				actualPodNames = append(actualPodNames, po.Namespace+"/"+po.Name)
			}
			sort.Strings(actualPodNames)
			sort.Strings(tt.expectedPods)
			Expect(actualPodNames).To(Equal(tt.expectedPods), ttName)

			By("checking runners")
			var actualRunnerNames []string
			for key := range tt.inputRunners {
				split := strings.Split(key, "/")
				owner := split[0]
				repo := split[1]
				runnerList, _ := githubClientFactory.ListRunners(ctx, owner, repo, nil)
				for _, runner := range runnerList {
					actualRunnerNames = append(actualRunnerNames, owner+"/"+repo+"/"+runner.Name)
				}
			}
			sort.Strings(actualRunnerNames)
			sort.Strings(tt.expectedRunners)
			Expect(actualRunnerNames).To(Equal(tt.expectedRunners), ttName)

			for _, rp := range tt.inputRunnerPools {
				By("stopping runnerpool manager; " + rp.Name)
				Expect(runnerManager.Stop(rp)).To(Succeed(), ttName)
				runnerList, _ := githubClientFactory.ListRunners(ctx, rp.GetOwner(), rp.GetRepository(), []string{rp.Name})
				Expect(runnerList).To(BeEmpty(), ttName)
			}

			By("tearing down")
			k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns1"))
			k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns2"))
			time.Sleep(500 * time.Millisecond)
		}
	})

	It("should remove pod-template-hash label from pods and create pdbs for the pods", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting runnerpool manager")
		rp := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		rp.Spec.Replicas = 5
		rp.Spec.MaxRunnerPods = 8
		runnerManager.StartOrUpdate(rp, nil)

		By("creating pods and runners")
		inputPods := []struct {
			spec  *corev1.Pod
			ip    string
			state string
		}{
			{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "running"}, // runner is idle. The label should not be removed from this pod.
			{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.2", state: "running"}, // runner is busy.
			{spec: makePod("pod3", "test-ns1", "rp1"), ip: "10.0.0.3", state: "running"}, // runner is busy.
			{spec: makePod("pod4", "test-ns1", "rp1"), ip: "10.0.0.4", state: "debugging"},
			{spec: makePod("pod5", "test-ns1", "rp1"), ip: "10.0.0.5", state: "debugging"},
		}
		inputRunners := []*github.Runner{
			{Name: "pod1", ID: 1, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
			{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
			{Name: "pod3", ID: 3, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
		}
		for _, inputPod := range inputPods {
			inputPod.spec.Labels["pod-template-hash"] = "foo"
			Expect(k8sClient.Create(ctx, inputPod.spec)).To(Succeed())
			created := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: inputPod.spec.Name, Namespace: inputPod.spec.Namespace}, created)).To(Succeed())
			created.Status.PodIP = inputPod.ip
			created.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(ctx, created)).To(Succeed())

			status := runner.Status{State: inputPod.state}
			if inputPod.state == "debugging" {
				t1 := time.Now()
				t2 := time.Now().Add(24 * time.Hour)
				status.FinishedAt = &t1
				status.DeletionTime = &t2
			}
			runnerPodClient.SetStatus(created.Status.PodIP, &status)
		}
		githubClientFactory.SetRunners(map[string][]*github.Runner{"owner/repo1": inputRunners})
		time.Sleep(2 * time.Second)

		By("checking pods")
		podList := new(corev1.PodList)
		Expect(k8sClient.List(ctx, podList, client.InNamespace("test-ns1"))).To(Succeed())

		var unlabeledPod *corev1.Pod
		var labeledPodNames []string
		for i := range podList.Items {
			po := &podList.Items[i]
			if _, ok := po.Labels["pod-template-hash"]; ok {
				labeledPodNames = append(labeledPodNames, po.Name)
			} else {
				unlabeledPod = po
			}
		}
		// In this test, only three (= MaxRunnerPods - Replicas) pods will be unlabeled. Other two pods will remain labeled.
		// It is undetermined which Pod other than pod1 will remain. It depends on enumeration order.
		Expect(labeledPodNames).To(HaveLen(len(inputPods) - int(rp.Spec.MaxRunnerPods-rp.Spec.Replicas)))
		Expect(labeledPodNames).To(ContainElement("pod1"))

		By("deleting one of the unlabeled pods")
		Expect(k8sClient.Delete(ctx, unlabeledPod)).To(Succeed())
		time.Sleep(2 * time.Second)

		By("checking pods")
		Expect(k8sClient.List(ctx, podList, client.InNamespace("test-ns1"), client.MatchingLabels{"pod-template-hash": "foo"})).To(Succeed())
		Expect(podList.Items).To(HaveLen(1))
		Expect(podList.Items[0].Name).To(Equal("pod1"))

		By("checking pdbs (there should be no pdbs)")
		pdbList := new(policyv1.PodDisruptionBudgetList)
		Expect(k8sClient.List(ctx, pdbList, client.InNamespace("test-ns1"))).To(Succeed())
		Expect(pdbList.Items).To(HaveLen(0))

		By("changing configuration to deny disruption")
		rp.Spec.DenyDisruption = true
		runnerManager.StartOrUpdate(rp, nil)
		time.Sleep(2 * time.Second)

		By("checking pdbs")
		var pdbNames []string
		Expect(k8sClient.List(ctx, pdbList, client.InNamespace("test-ns1"))).To(Succeed())
		for _, pdb := range pdbList.Items {
			Expect(pdb.Spec.Selector).NotTo(BeNil())
			Expect(pdb.Spec.Selector.MatchLabels["meows.cybozu.com/runner-pod-name"]).To(Equal(pdb.Name))
			Expect(pdb.Spec.MinAvailable).NotTo(BeNil())
			Expect(pdb.Spec.MinAvailable.Type).To(Equal(intstr.Int))
			Expect(pdb.Spec.MinAvailable.IntVal).To(Equal(int32(1)))
			Expect(pdb.OwnerReferences).To(HaveLen(1))
			Expect(pdb.OwnerReferences[0].Kind).To(Equal("Pod"))
			Expect(pdb.OwnerReferences[0].Name).To(Equal(pdb.Name))
			pdbNames = append(pdbNames, pdb.Name)
		}

		Expect(k8sClient.List(ctx, podList, client.InNamespace("test-ns1"))).To(Succeed())
		for _, pod := range podList.Items {
			if pod.Name == "pod1" {
				continue
			}
			Expect(pod.Labels["meows.cybozu.com/runner-pod-name"]).To(Equal(pod.Name))
			Expect(pdbNames).To(ContainElement(pod.Name))
		}

		By("tearing down")
		Expect(runnerManager.Stop(rp)).To(Succeed())
	})

	It("should expose metrics about runnerpools", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("checking metrics are not exposed")
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")

		By("creating rp1")
		rp1 := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		rp1.Spec.Replicas = 1
		runnerManager.StartOrUpdate(rp1, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("updating rp1")
		rp1.Spec.Replicas = 2
		runnerManager.StartOrUpdate(rp1, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
					"Value": BeNumerically("==", 2.0),
				})),
			}),
		)

		By("creating rp2")
		rp2 := makeRunnerPoolWithRepository("rp2", "test-ns2", "owner/repo1")
		rp2.Spec.Replicas = 1
		runnerManager.StartOrUpdate(rp2, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
					"Value": BeNumerically("==", 2.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp2")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("deleting rp1")
		Expect(runnerManager.Stop(rp1)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp2")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("deleting rp2")
		Expect(runnerManager.Stop(rp2)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
	})

	It("should expose metrics about runners (single runnerpool)", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating a runnerpool")
		rp1 := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		runnerManager.StartOrUpdate(rp1, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
				})),
			}),
		)
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")

		By("creating runner pods")
		dummyPods := []*corev1.Pod{
			makePod("pod1", "test-ns1", "rp1"),
			makePod("pod2", "test-ns1", "rp1"),
		}
		for _, po := range dummyPods {
			Expect(k8sClient.Create(ctx, po)).To(Succeed())
		}

		By("creating runners")
		runenrs := map[string][]*github.Runner{
			"owner/repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
			},
		}
		githubClientFactory.SetRunners(runenrs)
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)

		By("updating runners")
		runenrs = map[string][]*github.Runner{
			"owner/repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // metrics should not exist. "Offline" AND "Runner pod is not exist".
			},
		}
		githubClientFactory.SetRunners(runenrs)
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)

		By("deleting runnerpool")
		Expect(runnerManager.Stop(rp1)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")
	})

	It("should expose metrics about runners (some runnerpools)", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating runnerpools")
		rp1 := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		rp2 := makeRunnerPoolWithRepository("rp2", "test-ns1", "owner/repo1")
		rp3 := makeRunnerPoolWithRepository("rp3", "test-ns2", "owner/repo2")
		runnerManager.StartOrUpdate(rp1, nil)
		runnerManager.StartOrUpdate(rp2, nil)
		runnerManager.StartOrUpdate(rp3, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
				})),
				"1": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2")}),
				})),
				"2": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp3")}),
				})),
			}),
		)
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")

		By("creating runners")
		runenrs := map[string][]*github.Runner{
			"owner/repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp2"}},
			},
			"owner/repo2": {
				{Name: "pod4", ID: 4, Online: true, Busy: true, Labels: []string{"test-ns2/rp3"}},
				{Name: "pod5", ID: 5, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}}, // metrics should not exist.
				{Name: "pod6", ID: 6, Online: true, Busy: true, Labels: []string{}},               // metrics should not exist.
			},
		}
		githubClientFactory.SetRunners(runenrs)
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp3"), "runner": Equal("pod4")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp3"), "runner": Equal("pod4")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("updating runners")
		runenrs = map[string][]*github.Runner{
			"owner/repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: true, Busy: true, Labels: []string{"test-ns1/rp2"}},
			},
		}
		githubClientFactory.SetRunners(runenrs)
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("deleting runnerpool (1)")
		Expect(runnerManager.Stop(rp1)).To(Succeed())
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2")}),
				})),
				"1": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns2/rp3")}),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp2"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

		By("tearing down")
		Expect(runnerManager.Stop(rp2)).To(Succeed())
		Expect(runnerManager.Stop(rp3)).To(Succeed())
	})

	It("should expose metrics about job results", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("starting runnerpool manager")
		rp := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		runnerManager.StartOrUpdate(rp, nil)
		time.Sleep(1 * time.Second)
		By("checking metrics are initialized with zero values")
		CounterShouldHaveValue(metricsURL, "meows_runnerpool_job_results_total", ConsistOf(
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("success")}),
				"Value": BeNumerically("==", 0.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("failure")}),
				"Value": BeNumerically("==", 0.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("cancelled")}),
				"Value": BeNumerically("==", 0.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("unknown")}),
				"Value": BeNumerically("==", 0.0),
			})),
		))
		HistogramShouldHaveValue(metricsURL, "meows_runnerpool_job_duration_seconds", ConsistOf(
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("success")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("failure")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("cancelled")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("unknown")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
		))

		By("creating debugging pods")
		finishedAt := time.Now()
		deletionTime := finishedAt.Add(24 * time.Hour)
		startedAt := func(d time.Duration) *time.Time {
			t := finishedAt.Add(-d)
			return &t
		}
		inputPods := []struct {
			spec       *corev1.Pod
			ip         string
			result     string
			finishedAt time.Time
			jobInfo    *runner.JobInfo
		}{
			{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", result: "success", finishedAt: finishedAt, jobInfo: &runner.JobInfo{StartedAt: startedAt(10 * time.Minute)}},
			{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.2", result: "failure", finishedAt: finishedAt, jobInfo: &runner.JobInfo{StartedAt: startedAt(30 * time.Minute)}},
			{spec: makePod("pod3", "test-ns1", "rp1"), ip: "10.0.0.3", result: "unknown", finishedAt: finishedAt},                                                               // job-started was not called.
			{spec: makePod("pod4", "test-ns1", "rp1"), ip: "10.0.0.4", result: "invalid", finishedAt: finishedAt, jobInfo: &runner.JobInfo{StartedAt: startedAt(-time.Minute)}}, // invalid result and start time.
			{spec: makePod("pod5", "test-ns1", "rp1"), ip: "10.0.0.5", result: "success", finishedAt: finishedAt.Add(-time.Hour)},                                               // finished before the manager started.
		}
		for _, inputPod := range inputPods {
			Expect(k8sClient.Create(ctx, inputPod.spec)).To(Succeed())
			created := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: inputPod.spec.Name, Namespace: inputPod.spec.Namespace}, created)).To(Succeed())
			created.Status.PodIP = inputPod.ip
			created.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(ctx, created)).To(Succeed())

			podFinishedAt := inputPod.finishedAt
			runnerPodClient.SetStatus(inputPod.ip, &runner.Status{
				State:        "debugging",
				Result:       inputPod.result,
				FinishedAt:   &podFinishedAt,
				DeletionTime: &deletionTime,
				JobInfo:      inputPod.jobInfo,
			})
		}

		By("checking metrics are recorded once per job")
		time.Sleep(3 * time.Second)
		CounterShouldHaveValue(metricsURL, "meows_runnerpool_job_results_total", ConsistOf(
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("success")}),
				"Value": BeNumerically("==", 1.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("failure")}),
				"Value": BeNumerically("==", 1.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("cancelled")}),
				"Value": BeNumerically("==", 0.0),
			})),
			PointTo(MatchAllFields(Fields{
				"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("unknown")}),
				"Value": BeNumerically("==", 2.0),
			})),
		))
		HistogramShouldHaveValue(metricsURL, "meows_runnerpool_job_duration_seconds", ConsistOf(
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("success")}),
				"SampleCount": BeNumerically("==", 1),
				"SampleSum":   BeNumerically("~", 600, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("failure")}),
				"SampleCount": BeNumerically("==", 1),
				"SampleSum":   BeNumerically("~", 1800, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("cancelled")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
			PointTo(MatchAllFields(Fields{
				"Label":       MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "result": Equal("unknown")}),
				"SampleCount": BeNumerically("==", 0),
				"SampleSum":   BeNumerically("~", 0, 1),
			})),
		))

		By("deleting runnerpool")
		Expect(runnerManager.Stop(rp)).To(Succeed())
		time.Sleep(2 * time.Second)
		CounterShouldNotExist(metricsURL, "meows_runnerpool_job_results_total")
		HistogramShouldNotExist(metricsURL, "meows_runnerpool_job_duration_seconds")
	})

	It("should delete all runners and metrics", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory()
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, scheme, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating a runnerpool")
		rp1 := makeRunnerPoolWithRepository("rp1", "test-ns1", "owner/repo1")
		runnerManager.StartOrUpdate(rp1, nil)
		time.Sleep(2 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runnerpool_replicas",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchFields(IgnoreExtras, Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1")}),
				})),
			}),
		)
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")

		By("creating runner pods")
		dummyPods := []*corev1.Pod{
			makePod("pod1", "test-ns1", "rp1"),
			makePod("pod2", "test-ns1", "rp1"),
		}
		for _, po := range dummyPods {
			Expect(k8sClient.Create(ctx, po)).To(Succeed())
		}

		By("creating runners")
		runenrs := map[string][]*github.Runner{
			"owner/repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
			},
		}
		githubClientFactory.SetRunners(runenrs)
		time.Sleep(3 * time.Second)
		MetricsShouldHaveValue(metricsURL, "meows_runner_online",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		MetricsShouldHaveValue(metricsURL, "meows_runner_busy",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod1")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod2")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("test-ns1/rp1"), "runner": Equal("pod3")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)

		By("deleting runnerpool")
		Expect(runnerManager.Stop(rp1)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")
		runnerList, _ := githubClientFactory.ListRunners(ctx, rp1.GetOwner(), rp1.GetRepository(), nil)
		Expect(runnerList).To(BeEmpty())
	})
})

func MetricsShouldNotExist(url, name string) {
	_, err := metrics.FetchGauge(context.Background(), url, name)
	ExpectWithOffset(1, err).Should(MatchError(metrics.ErrNotExist))
}

func MetricsShouldHaveValue(url, name string, matcher gomegatypes.GomegaMatcher) {
	m, err := metrics.FetchGauge(context.Background(), url, name)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, m).To(matcher)
}

func CounterShouldNotExist(url, name string) {
	_, err := metrics.FetchCounter(context.Background(), url, name)
	ExpectWithOffset(1, err).Should(MatchError(metrics.ErrNotExist))
}

func CounterShouldHaveValue(url, name string, matcher gomegatypes.GomegaMatcher) {
	m, err := metrics.FetchCounter(context.Background(), url, name)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, m).To(matcher)
}

func HistogramShouldNotExist(url, name string) {
	_, err := metrics.FetchHistogram(context.Background(), url, name)
	ExpectWithOffset(1, err).Should(MatchError(metrics.ErrNotExist))
}

func HistogramShouldHaveValue(url, name string, matcher gomegatypes.GomegaMatcher) {
	m, err := metrics.FetchHistogram(context.Background(), url, name)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, m).To(matcher)
}
