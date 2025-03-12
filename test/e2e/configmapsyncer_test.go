package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	syncv1alpha1 "github.com/devShahriar/configmap-sync-controller/api/v1alpha1"
	"github.com/devShahriar/configmap-sync-controller/test/utils"
)

var _ = Describe("ConfigMapSyncer", Ordered, func() {
	const (
		sourceNamespace     = "default"
		targetNamespace1    = "app1"
		targetNamespace2    = "app2"
		monitoringNamespace = "monitoring"
		sourceConfigMapName = "source-config"
		syncerName          = "test-syncer"
		syncInterval        = 3 // seconds
	)

	BeforeAll(func() {
		// Clean up any existing resources from previous test runs
		By("cleaning up any existing test resources")
		cleanup := func() {
			utils.Run(
				exec.Command(
					"kubectl",
					"delete",
					"namespace",
					"app1",
					"--ignore-not-found=true",
				),
			)
			utils.Run(
				exec.Command(
					"kubectl",
					"delete",
					"namespace",
					"app2",
					"--ignore-not-found=true",
				),
			)
			utils.Run(
				exec.Command(
					"kubectl",
					"delete",
					"namespace",
					"monitoring",
					"--ignore-not-found=true",
				),
			)
			utils.Run(
				exec.Command(
					"kubectl",
					"delete",
					"configmap",
					"source-config",
					"-n",
					"default",
					"--ignore-not-found=true",
				),
			)
			utils.Run(
				exec.Command(
					"kubectl",
					"delete",
					"configmapsyncer",
					"test-syncer",
					"-n",
					"default",
					"--ignore-not-found=true",
				),
			)
		}

		cleanup()

		// Wait for resources to be fully deleted
		time.Sleep(5 * time.Second)

		By("waiting for controller to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command(
				"kubectl",
				"get",
				"pods",
				"-n",
				"configmap-sync-controller-system",
				"-l",
				"control-plane=controller-manager",
				"-o",
				"jsonpath={.items[0].status.phase}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Running"))
		}, "60s", "5s").Should(Succeed())

		By("creating test namespaces")
		createNamespace := func(ns string) {
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "create", "namespace", ns)
				_, err := utils.Run(cmd)
				g.Expect(err).
					NotTo(HaveOccurred(), fmt.Sprintf("Failed to create namespace %s", ns))
			}, "30s", "5s").Should(Succeed())
		}

		createNamespace(targetNamespace1)
		createNamespace(targetNamespace2)
		createNamespace(monitoringNamespace)

		// Create source ConfigMap
		sourceConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sourceConfigMapName,
				Namespace: sourceNamespace,
			},
			Data: map[string]string{
				"app.properties": "log.level=INFO\nmax.connections=100\ntimeout=30s",
			},
		}
		configMapYAML, err := yaml.Marshal(sourceConfigMap)
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(string(configMapYAML))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create source ConfigMap")

		// Create ConfigMapSyncer
		syncer := &syncv1alpha1.ConfigMapSyncer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      syncerName,
				Namespace: sourceNamespace,
			},
			Spec: syncv1alpha1.ConfigMapSyncerSpec{
				MasterConfigMap: syncv1alpha1.ConfigMapReference{
					Name:      sourceConfigMapName,
					Namespace: sourceNamespace,
				},
				TargetNamespaces: []string{
					targetNamespace1,
					targetNamespace2,
					monitoringNamespace,
				},
				MergeStrategy: "Merge",
				SyncInterval:  syncInterval,
			},
		}
		syncerYAML, err := yaml.Marshal(syncer)
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(string(syncerYAML))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ConfigMapSyncer")
	})

	AfterAll(func() {
		// Cleanup
		for _, ns := range []string{targetNamespace1, targetNamespace2, monitoringNamespace} {
			cmd := exec.Command("kubectl", "delete", "namespace", ns)
			_, _ = utils.Run(cmd)
		}
		cmd := exec.Command(
			"kubectl",
			"delete",
			"configmap",
			sourceConfigMapName,
			"-n",
			sourceNamespace,
		)
		_, _ = utils.Run(cmd)
		cmd = exec.Command(
			"kubectl",
			"delete",
			"configmapsyncer",
			syncerName,
			"-n",
			sourceNamespace,
		)
		_, _ = utils.Run(cmd)
	})

	Context("Sync Functionality", func() {
		It("should sync ConfigMap to target namespaces", func() {
			// Wait for initial sync
			time.Sleep(time.Duration(syncInterval+2) * time.Second)

			By("verifying ConfigMaps are synced to all target namespaces")
			for _, ns := range []string{targetNamespace1, targetNamespace2, monitoringNamespace} {
				verifyConfigMapSync := func(g Gomega) {
					cmd := exec.Command(
						"kubectl",
						"get",
						"configmap",
						sourceConfigMapName,
						"-n",
						ns,
						"-o",
						"jsonpath={.data.app\\.properties}",
					)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).
						To(Equal("log.level=INFO\nmax.connections=100\ntimeout=30s"))
				}
				Eventually(
					verifyConfigMapSync,
					time.Duration(syncInterval*2)*time.Second,
				).Should(Succeed())
			}
		})

		It("should sync updates from source ConfigMap", func() {
			By("updating source ConfigMap")
			updatedData := "log.level=DEBUG\nmax.connections=200\ntimeout=60s"
			cmd := exec.Command("kubectl", "patch", "configmap", sourceConfigMapName,
				"-n", sourceNamespace,
				"--type=merge",
				"-p", fmt.Sprintf(`{"data":{"app.properties":"%s"}}`, updatedData))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to update source ConfigMap")

			By("verifying updates are synced to target namespaces")
			for _, ns := range []string{targetNamespace1, targetNamespace2, monitoringNamespace} {
				verifyConfigMapUpdate := func(g Gomega) {
					cmd := exec.Command(
						"kubectl",
						"get",
						"configmap",
						sourceConfigMapName,
						"-n",
						ns,
						"-o",
						"jsonpath={.data.app\\.properties}",
					)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal(updatedData))
				}
				Eventually(
					verifyConfigMapUpdate,
					time.Duration(syncInterval*2)*time.Second,
				).Should(Succeed())
			}
		})

		It("should handle merge strategy correctly", func() {
			By("adding new data to source ConfigMap")
			cmd := exec.Command("kubectl", "patch", "configmap", sourceConfigMapName,
				"-n", sourceNamespace,
				"--type=merge",
				"-p", `{"data":{"new.property":"value1"}}`)
			_, err := utils.Run(cmd)
			Expect(
				err,
			).NotTo(HaveOccurred(), "Failed to add new data to source ConfigMap")

			By("verifying new data is merged in target namespaces")
			for _, ns := range []string{targetNamespace1, targetNamespace2, monitoringNamespace} {
				verifyMerge := func(g Gomega) {
					cmd := exec.Command(
						"kubectl",
						"get",
						"configmap",
						sourceConfigMapName,
						"-n",
						ns,
						"-o",
						"jsonpath={.data.new\\.property}",
					)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("value1"))
				}
				Eventually(
					verifyMerge,
					time.Duration(syncInterval*2)*time.Second,
				).Should(Succeed())
			}
		})
	})
})
