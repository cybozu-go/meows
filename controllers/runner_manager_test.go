package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
			state        string
			finishedAt   time.Time
			deletionTime time.Time
		}
		testCases := []struct {
			name             string
			inputRunnerPools []*meowsv1alpha1.RunnerPool
			inputPods        []*inputPod
			inputRunners     map[string][]*github.Runner // key: "<Repository name>"
			expectedPods     []string                    // slice of "<Namespace>/<Pod name>"
			expectedRunners  []string                    // slice of "<Repository name>/<Runner name>"
		}{
			{
				name: "delete pods",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPool("rp1", "test-ns1", "repo1"),
					makeRunnerPool("rp2", "test-ns1", "repo2"),
					makeRunnerPoolWithRecreateDeadline("rp3", "test-ns2", "repo2", "5s"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "debugging", finishedAt: time.Now(), deletionTime: time.Now()}, // state is debugging.
					{spec: makePod("pod2", "test-ns1", "rp2"), ip: "10.0.0.2", state: "stale"},                                                       // state is stale.
					{spec: makePod("pod3", "test-ns2", "rp3"), ip: "10.0.0.3", state: "running"},                                                     // recreate deadline is exceeded and runner is not exist.
					{spec: makePod("pod4", "test-ns2", "rp3"), ip: "10.0.0.4", state: "running"},                                                     // recreate deadline is exceeded and runner is not busy.
				},
				inputRunners: map[string][]*github.Runner{
					"repo2": {
						{Name: "pod4", ID: 4, Online: true, Busy: false, Labels: []string{"test-ns2/rp3"}},
					},
				},
				expectedPods: nil,
				expectedRunners: []string{
					"repo2/pod4",
				},
			},
			{
				name: "should not delete pods",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPool("rp1", "test-ns1", "repo1"),
					makeRunnerPool("rp2", "test-ns1", "repo2"),
					makeRunnerPoolWithRecreateDeadline("rp3", "test-ns2", "repo2", "5s"),
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
					"repo2": {
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
					"repo2/pod2",
				},
			},
			{
				name: "delete runners",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPool("rp1", "test-ns1", "repo1"),
					makeRunnerPool("rp2", "test-ns1", "repo2"),
				},
				inputPods: nil,
				inputRunners: map[string][]*github.Runner{
					"repo1": {
						{Name: "pod1", ID: 1, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod does not exist, offline
						{Name: "pod2", ID: 2, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod does not exist, offline
					},
					"repo2": {
						{Name: "pod3", ID: 3, Online: false, Busy: false, Labels: []string{"test-ns1/rp2"}}, // pod does not exist, offline
					},
				},
				expectedPods:    nil,
				expectedRunners: nil,
			},
			{
				name: "should not delete runners",
				inputRunnerPools: []*meowsv1alpha1.RunnerPool{
					makeRunnerPool("rp1", "test-ns1", "repo1"),
					makeRunnerPool("rp2", "test-ns1", "repo2"),
				},
				inputPods: []*inputPod{
					{spec: makePod("pod1", "test-ns1", "rp1"), ip: "10.0.0.1", state: "running"},
					{spec: makePod("pod2", "test-ns1", "rp1"), ip: "10.0.0.2", state: "running"},
				},
				inputRunners: map[string][]*github.Runner{
					"repo1": {
						{Name: "pod1", ID: 1, Online: false, Busy: false, Labels: []string{"test-ns1/rp1"}}, // pod exists
						{Name: "pod2", ID: 2, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},   // pod exists
						{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},  // pod does not exist, but online
					},
					"repo2": {
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
					"repo1/pod1",
					"repo1/pod2",
					"repo1/pod3",
					"repo2/pod1",
					"repo2/pod2",
					"repo2/pod3",
				},
			},
		}

		for _, tt := range testCases {
			By(tt.name)
			ttName := fmt.Sprintf("test case name is '%s'", tt.name)

			By("preparing fake clients")
			runnerPodClient := runner.NewFakeClient()
			githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
			runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

			By("preparing pods and runners")
			for _, inputPod := range tt.inputPods {
				Expect(k8sClient.Create(ctx, inputPod.spec)).To(Succeed(), ttName)
				created := &corev1.Pod{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: inputPod.spec.Name, Namespace: inputPod.spec.Namespace}, created)).To(Succeed(), ttName)
				created.Status.PodIP = inputPod.ip
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
				runnerManager.StartOrUpdate(ctx, rp, nil)
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
			for repo := range tt.inputRunners {
				runnerList, _ := githubClientFactory.ListRunners(ctx, repo, nil)
				for _, runner := range runnerList {
					actualRunnerNames = append(actualRunnerNames, repo+"/"+runner.Name)
				}
			}
			sort.Strings(actualRunnerNames)
			sort.Strings(tt.expectedRunners)
			Expect(actualRunnerNames).To(Equal(tt.expectedRunners), ttName)

			for _, rp := range tt.inputRunnerPools {
				By("stopping runnerpool manager; " + rp.Name)
				Expect(runnerManager.Stop(ctx, rp)).To(Succeed(), ttName)
				runnerList, _ := githubClientFactory.ListRunners(ctx, rp.Spec.RepositoryName, []string{rp.Name})
				Expect(runnerList).To(BeEmpty(), ttName)
			}

			By("tearing down")
			k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns1"))
			k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("test-ns2"))
			time.Sleep(500 * time.Millisecond)
		}
	})

	It("should remove pod-template-hash label from pods", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

		By("starting runnerpool manager")
		rp := makeRunnerPool("rp1", "test-ns1", "repo1")
		rp.Spec.Replicas = 5
		rp.Spec.MaxRunnerPods = 8
		runnerManager.StartOrUpdate(ctx, rp, nil)

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
		githubClientFactory.SetRunners(map[string][]*github.Runner{"repo1": inputRunners})
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
		Expect(labeledPodNames).To(HaveLen(2))
		Expect(labeledPodNames).To(ContainElement("pod1"))

		By("deleting one of the unlabeled pods")
		Expect(k8sClient.Delete(ctx, unlabeledPod)).To(Succeed())
		time.Sleep(2 * time.Second)

		By("checking pods")
		Expect(k8sClient.List(ctx, podList, client.InNamespace("test-ns1"), client.MatchingLabels{"pod-template-hash": "foo"})).To(Succeed())
		Expect(podList.Items).To(HaveLen(1))
		Expect(podList.Items[0].Name).To(Equal("pod1"))

		By("tearing down")
		Expect(runnerManager.Stop(ctx, rp)).To(Succeed())
	})

	It("should expose metrics about runnerpools", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

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
		rp1 := makeRunnerPool("rp1", "test-ns1", "repo1")
		rp1.Spec.Replicas = 1
		runnerManager.StartOrUpdate(ctx, rp1, nil)
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
		runnerManager.StartOrUpdate(ctx, rp1, nil)
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
		rp2 := makeRunnerPool("rp2", "test-ns2", "repo1")
		rp2.Spec.Replicas = 1
		runnerManager.StartOrUpdate(ctx, rp2, nil)
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
		Expect(runnerManager.Stop(ctx, rp1)).To(Succeed())
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
		Expect(runnerManager.Stop(ctx, rp2)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
	})

	It("should expose metrics about runners (single runnerpool)", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating a runnerpool")
		rp1 := makeRunnerPool("rp1", "test-ns1", "repo1")
		runnerManager.StartOrUpdate(ctx, rp1, nil)
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
			"repo1": {
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
			"repo1": {
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
		Expect(runnerManager.Stop(ctx, rp1)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")
	})

	It("should expose metrics about runners (some runnerpools)", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating runnerpools")
		rp1 := makeRunnerPool("rp1", "test-ns1", "repo1")
		rp2 := makeRunnerPool("rp2", "test-ns1", "repo1")
		rp3 := makeRunnerPool("rp3", "test-ns2", "repo2")
		runnerManager.StartOrUpdate(ctx, rp1, nil)
		runnerManager.StartOrUpdate(ctx, rp2, nil)
		runnerManager.StartOrUpdate(ctx, rp3, nil)
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
			"repo1": {
				{Name: "pod1", ID: 1, Online: true, Busy: true, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod2", ID: 2, Online: true, Busy: false, Labels: []string{"test-ns1/rp1"}},
				{Name: "pod3", ID: 3, Online: true, Busy: false, Labels: []string{"test-ns1/rp2"}},
			},
			"repo2": {
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
			"repo1": {
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
		Expect(runnerManager.Stop(ctx, rp1)).To(Succeed())
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
		Expect(runnerManager.Stop(ctx, rp2)).To(Succeed())
		Expect(runnerManager.Stop(ctx, rp3)).To(Succeed())
	})

	It("should delete all runners and metrics", func() {
		By("preparing fake clients")
		runnerPodClient := runner.NewFakeClient()
		githubClientFactory := github.NewFakeClientFactory("runnermanager-org")
		runnerManager := NewRunnerManager(ctrl.Log, k8sClient, githubClientFactory, runnerPodClient, time.Second)

		By("starting metrics server")
		server := &http.Server{Addr: metricsPort, Handler: promhttp.Handler()}
		go func() {
			server.ListenAndServe()
		}()
		defer server.Shutdown(context.Background())
		time.Sleep(1 * time.Second)

		By("creating a runnerpool")
		rp1 := makeRunnerPool("rp1", "test-ns1", "repo1")
		runnerManager.StartOrUpdate(ctx, rp1, nil)
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
			"repo1": {
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
		Expect(runnerManager.Stop(ctx, rp1)).To(Succeed())
		time.Sleep(2 * time.Second)
		MetricsShouldNotExist(metricsURL, "meows_runnerpool_replicas")
		MetricsShouldNotExist(metricsURL, "meows_runner_online")
		MetricsShouldNotExist(metricsURL, "meows_runner_busy")
		runnerList, _ := githubClientFactory.ListRunners(ctx, "repo1", nil)
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
