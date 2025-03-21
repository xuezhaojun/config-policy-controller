// Copyright (c) 2023 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package e2e

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"open-cluster-management.io/config-policy-controller/test/utils"
)

var _ = Describe("Test compliance events of enforced policies that define a status", Serial, func() {
	const (
		rsrcPath      = "../resources/case34_enforce_w_status/"
		policyYAML    = rsrcPath + "policy.yaml"
		policyName    = "case34-parent"
		cfgPlcYAML    = rsrcPath + "config-policy.yaml"
		updatedCfgPlc = rsrcPath + "config-policy-updated.yaml"
		cfgPlcName    = "case34-cfgpol"
		nsName        = "case34-ns"
		finalizerName = "policy.open-cluster-management.io/stuck-test"
	)

	It("Should have the expected events", func() {
		By("Setting up the policy")
		createObjWithParent(policyYAML, policyName, cfgPlcYAML, testNamespace, gvrPolicy, gvrConfigPolicy)

		By("Checking there is a NonCompliant event on the policy")
		Eventually(func() interface{} {
			return utils.GetMatchingEvents(clientManaged, testNamespace,
				policyName, cfgPlcName, "^NonCompliant;", defaultTimeoutSeconds)
		}, defaultTimeoutSeconds, 5).ShouldNot(BeEmpty())

		By("Checking there are no Compliant events on the policy")
		Consistently(func() interface{} {
			return utils.GetMatchingEvents(clientManaged, testNamespace,
				policyName, cfgPlcName, "^Compliant;", defaultTimeoutSeconds)
		}, defaultConsistentlyDuration, 5).Should(BeEmpty())

		By("Updating the policy")
		utils.Kubectl("apply", "-f", updatedCfgPlc, "-n", testNamespace)

		By("Checking there are no Compliant events created during the update flow")
		Consistently(func() interface{} {
			return utils.GetMatchingEvents(clientManaged, testNamespace,
				policyName, cfgPlcName, "^Compliant;", defaultTimeoutSeconds)
		}, defaultConsistentlyDuration, 5).Should(BeEmpty())

		By("Setting a finalizer on the namespace")
		utils.Kubectl("patch", "ns", nsName, "--type=merge",
			`-p={"metadata":{"finalizers":["`+finalizerName+`"]}}`)
		Eventually(func(g Gomega) []string {
			ns := utils.GetClusterLevelWithTimeout(clientManagedDynamic, gvrNS,
				nsName, true, defaultTimeoutSeconds)
			g.Expect(ns).ShouldNot(BeNil())

			return ns.GetFinalizers()
		}, defaultTimeoutSeconds, 2).Should(ContainElement(finalizerName))

		By("Marking the namespace for deletion")
		utils.Kubectl("delete", "ns", nsName, "--wait=false")

		By("Checking there is now a Compliant event on the policy")
		Eventually(func() interface{} {
			return utils.GetMatchingEvents(clientManaged, testNamespace,
				policyName, cfgPlcName, "^Compliant;", defaultTimeoutSeconds)
		}, defaultTimeoutSeconds, 5).ShouldNot(BeEmpty())
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			events := utils.GetMatchingEvents(clientManaged, testNamespace,
				policyName, ".*", ".*", defaultTimeoutSeconds)

			By("Test failed, printing compliance events for debugging, event count = " + strconv.Itoa(len(events)))
			for _, ev := range events {
				GinkgoWriter.Println("---")
				GinkgoWriter.Println("Name:", ev.Name)
				GinkgoWriter.Println("Reason:", ev.Reason)
				GinkgoWriter.Println("Message:", ev.Message)
				GinkgoWriter.Println("FirstTimestamp:", ev.FirstTimestamp)
				GinkgoWriter.Println("LastTimestamp:", ev.LastTimestamp)
				GinkgoWriter.Println("Count:", ev.Count)
				GinkgoWriter.Println("Type:", ev.Type)
				GinkgoWriter.Println("---")
			}
		}

		utils.KubectlDelete("policy", policyName, "-n", "managed")
		configPlc := utils.GetWithTimeout(clientManagedDynamic, gvrConfigPolicy,
			cfgPlcName, "managed", false, defaultTimeoutSeconds,
		)
		Expect(configPlc).To(BeNil())

		utils.Kubectl("patch", "ns", nsName, "--type=merge", `-p={"metadata":{"finalizers":[]}}`)
		utils.KubectlDelete("ns", nsName)
		utils.KubectlDelete("event", "--field-selector=involvedObject.name="+policyName, "-n", "managed")
		utils.KubectlDelete("event", "--field-selector=involvedObject.name="+cfgPlcName, "-n", "managed")
	})
})
