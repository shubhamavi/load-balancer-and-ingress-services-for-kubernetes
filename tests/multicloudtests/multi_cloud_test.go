/*
 * Copyright 2019-2020 VMware, Inc.
 * All Rights Reserved.
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*   http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/
package k8stest

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	crdfake "github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/internal/client/clientset/versioned/fake"
	"github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/internal/k8s"
	"github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/tests/integrationtest"

	corev1 "k8s.io/api/core/v1"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	// To Do: add test for openshift route
	//oshiftfake "github.com/openshift/client-go/route/clientset/versioned/fake"

	"github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

const defaultMockFilePath = "../avimockobjects"
const invalidFilePath = "invalidmock"

var kubeClient *k8sfake.Clientset
var crdClient *crdfake.Clientset
var dynamicClient *dynamicfake.FakeDynamicClient
var keyChan chan string
var ctrl *k8s.AviController
var RegisteredInformers = []string{
	utils.ServiceInformer,
	utils.EndpointInformer,
	utils.IngressInformer,
	utils.SecretInformer,
	utils.NSInformer,
	utils.NodeInformer,
	utils.ConfigMapInformer,
}

func syncFuncForTest(key string, wg *sync.WaitGroup) error {
	keyChan <- key
	return nil
}

// empty key ("") means we are not expecting the key
func waitAndverify(t *testing.T, key string) {
	waitChan := make(chan int)
	go func() {
		time.Sleep(20 * time.Second)
		waitChan <- 1
	}()

	select {
	case data := <-keyChan:
		if key == "" {
			t.Fatalf("unpexpected key: %v", data)
		} else if data != key {
			t.Fatalf("error in match expected: %v, got: %v", key, data)
		}
	case _ = <-waitChan:
		if key != "" {
			t.Fatalf("timed out waiting for %v", key)
		}
	}
}

func setupQueue(stopCh <-chan struct{}) {
	ingestionQueue := utils.SharedWorkQueue().GetQueueByName(utils.ObjectIngestionLayer)
	wgIngestion := &sync.WaitGroup{}

	ingestionQueue.SyncFunc = syncFuncForTest
	ingestionQueue.Run(stopCh, wgIngestion)
}

func AddConfigMap(t *testing.T) {
	aviCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "avi-system",
			Name:      "avi-k8s-config",
		},
	}
	_, err := kubeClient.CoreV1().ConfigMaps("avi-system").Create(aviCM)
	if err != nil {
		t.Fatalf("error in adding configmap: %v", err)
	}
	time.Sleep(10 * time.Second)
}

func AddCMap() {
    aviCM := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Namespace: "avi-system",
            Name:      "avi-k8s-config",
        },
    }
    kubeClient.CoreV1().ConfigMaps("avi-system").Create(aviCM)
}

func DeleteConfigMap(t *testing.T) {
	options := metav1.DeleteOptions{}
	err := kubeClient.CoreV1().ConfigMaps("avi-system").Delete("avi-k8s-config", &options)
	if err != nil {
		t.Fatalf("error in deleting configmap: %v", err)
	}
	time.Sleep(10 * time.Second)
}

func ValidateIngress(t *testing.T) {

	// validate svc first
	svcExample := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "red-ns",
			Name:      "testsvc",
		},
	}
	_, err := kubeClient.CoreV1().Services("red-ns").Create(svcExample)
	if err != nil {
		t.Fatalf("error in adding Service: %v", err)
	}
	waitAndverify(t, "L4LBService/red-ns/testsvc")

	ingrExample := &extensionv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "red-ns",
			Name:      "testingr",
		},
		Spec: extensionv1beta1.IngressSpec{
			Backend: &extensionv1beta1.IngressBackend{
				ServiceName: "testsvc",
			},
		},
	}
	_, err = kubeClient.ExtensionsV1beta1().Ingresses("red-ns").Create(ingrExample)
	if err != nil {
		t.Fatalf("error in adding Ingress: %v", err)
	}
	waitAndverify(t, "Ingress/red-ns/testingr")
	// delete svc and ingress
	err = kubeClient.CoreV1().Services("red-ns").Delete("testsvc", &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("error in deleting Service: %v", err)
	}
	waitAndverify(t, "L4LBService/red-ns/testsvc")
	err = kubeClient.ExtensionsV1beta1().Ingresses("red-ns").Delete("testingr", &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("error in adding Ingress: %v", err)
	}
	waitAndverify(t, "Ingress/red-ns/testingr")

}

func ValidateNode(t *testing.T) {
	nodeExample := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testnode",
			ResourceVersion: "1",
		},
		Spec: corev1.NodeSpec{
			PodCIDR: "10.244.0.0/24",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    "InternalIP",
					Address: "10.1.1.2",
				},
			},
		},
	}
	_, err := kubeClient.CoreV1().Nodes().Create(nodeExample)
	if err != nil {
		t.Fatalf("error in adding Node: %v", err)
	}
	waitAndverify(t, utils.NodeObj+"/testnode")

	nodeExample.ObjectMeta.ResourceVersion = "2"
	nodeExample.Spec.PodCIDR = "10.230.0.0/24"
	_, err = kubeClient.CoreV1().Nodes().Update(nodeExample)
	if err != nil {
		t.Fatalf("error in updating Node: %v", err)
	}
	waitAndverify(t, utils.NodeObj+"/testnode")

	err = kubeClient.CoreV1().Nodes().Delete("testnode", nil)
	if err != nil {
		t.Fatalf("error in Deleting Node: %v", err)
	}
	waitAndverify(t, utils.NodeObj+"/testnode")
}

func injectMWForCloud() {
	integrationtest.AddMiddleware(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.EscapedPath()
		if r.Method == "GET" && strings.Contains(url, "/api/cloud/") {
			integrationtest.FeedMockCollectionData(w, r, invalidFilePath)

		} else if r.Method == "GET" {
			integrationtest.FeedMockCollectionData(w, r, defaultMockFilePath)

		} else if strings.Contains(url, "login") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": "true"}`))
		}
	})
}

func TestMain(m *testing.M) {
	os.Setenv("INGRESS_API", "extensionv1")
	os.Setenv("NETWORK_NAME", "net123")
	os.Setenv("CLUSTER_NAME", "cluster")
	os.Setenv("SEG_NAME", "Default-Group")
	os.Setenv("NODE_NETWORK_LIST", `[{"networkName":"net123","cidrs":["10.79.168.0/22"]}]`)
	os.Setenv("CLOUD_NAME", "CLOUD_AWS")
	utils.SetCloudName("CLOUD_AWS")
	os.Setenv("SERVICE_TYPE", "NodePort")
	kubeClient = k8sfake.NewSimpleClientset()
	dynamicClient = dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	crdClient = crdfake.NewSimpleClientset()
	lib.SetCRDClientset(crdClient)
	utils.NewInformers(utils.KubeClientIntf{kubeClient}, RegisteredInformers)
	k8s.NewCRDInformers(crdClient)

	integrationtest.InitializeFakeAKOAPIServer()
	integrationtest.NewAviFakeClientInstance()
	defer integrationtest.AviFakeClientInstance.Close()

	ctrl = k8s.SharedAviController()
	stopCh := utils.SetupSignalHandler()
	ctrl.Start(stopCh)
	keyChan = make(chan string)
	ctrlCh := make(chan struct{})
	quickSyncCh := make(chan struct{})
        AddCMap()
	ctrl.HandleConfigMap(k8s.K8sinformers{Cs: kubeClient, DynamicClient: dynamicClient}, ctrlCh, stopCh, quickSyncCh)
	ctrl.SetupEventHandlers(k8s.K8sinformers{Cs: kubeClient, DynamicClient: dynamicClient})
	setupQueue(stopCh)
	os.Exit(m.Run())
}

// Cloud does not have a ipam_provider_ref configured, sync should be disabled
func TestVcenterCloudNoIpamDuringBootup(t *testing.T) {
        DeleteConfigMap(t)
	os.Setenv("CLOUD_NAME", "CLOUD_VCENTER")
	utils.SetCloudName("CLOUD_VCENTER")
	os.Setenv("SERVICE_TYPE", "ClusterIP")
	injectMWForCloud()

	AddConfigMap(t)

	if !ctrl.DisableSync {
		t.Fatalf("Validation for vcenter Cloud for ipam_provider_ref failed")
	}
	integrationtest.ResetMiddleware()
	DeleteConfigMap(t)
}

// TestAWSCloudValidation tests validation in place for public clouds
func TestAWSCloudValidation(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AWS")
	utils.SetCloudName("CLOUD_AWS")
	os.Setenv("SERVICE_TYPE", "NodePort")
	os.Setenv("NETWORK_NAME", "")

	AddConfigMap(t)

	if !ctrl.DisableSync {
		t.Fatalf("CLOUD_AWS should not be allowed if NETWORK_NAME is empty")
	}
	DeleteConfigMap(t)
	os.Setenv("NETWORK_NAME", "net123")
}

// TestAzureCloudValidation tests validation in place for public clouds
func TestAzureCloudValidation(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AZURE")
	utils.SetCloudName("CLOUD_AZURE")
	os.Setenv("SERVICE_TYPE", "NodePort")
	os.Setenv("NETWORK_NAME", "")

	AddConfigMap(t)

	if !ctrl.DisableSync {
		t.Fatalf("CLOUD_AZURE should not be allowed if NETWORK_NAME is empty")
	}
	DeleteConfigMap(t)
	os.Setenv("NETWORK_NAME", "net123")
}

// TestAWSCloudInClusterIPMode tests case where AWS CLOUD is configured in ClousterIP mode. Sync should be allowed.
func TestAWSCloudInClusterIPMode(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AWS")
	utils.SetCloudName("CLOUD_AWS")
	os.Setenv("SERVICE_TYPE", "ClusterIP")

	AddConfigMap(t)

	if ctrl.DisableSync {
		t.Fatalf("CLOUD_AWS should be allowed in ClusterIP mode")
	}
	DeleteConfigMap(t)
}

// TestAzureCloudInClusterIPMode tests case where Azure cloud is configured in ClusterIP mode. Sync should be allowed.
func TestAzureCloudInClusterIPMode(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AZURE")
	utils.SetCloudName("CLOUD_AZURE")
	os.Setenv("SERVICE_TYPE", "ClusterIP")

	AddConfigMap(t)

	if ctrl.DisableSync {
		t.Fatalf("CLOUD_AZURE should be allowed in ClusterIP mode")
	}
	DeleteConfigMap(t)

}

// TestGCPCloudInClusterIPMode tests case where GCP cloud is configured in ClusterIP mode. Sync should be allowed.
func TestGCPCloudInClusterIPMode(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_GCP")
	utils.SetCloudName("CLOUD_GCP")
	os.Setenv("SERVICE_TYPE", "ClusterIP")

	AddConfigMap(t)

	if ctrl.DisableSync {
		t.Fatalf("CLOUD_GCP should  be allowed in ClusterIP mode")
	}
	DeleteConfigMap(t)

}

// TestAzureCloudInNodePortMode tests case where Azure cloud is configured in NodePort mode. Sync should be enabled.
func TestAzureCloudInNodePortMode(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AZURE")
	utils.SetCloudName("CLOUD_AZURE")
	os.Setenv("SERVICE_TYPE", "NodePort")

	// add the config and check if the sync is enabled
	AddConfigMap(t)

	if ctrl.DisableSync {
		t.Fatalf("CLOUD_AZURE should be allowed in ClusterIP mode")
	}
	// validate the config
	ValidateIngress(t)
	ValidateNode(t)

	DeleteConfigMap(t)

}

// TestAWSCloudInNodePortMode tests case where AWS cloud is configured in NodePort mode. Sync should be enabled.
func TestAWSCloudInNodePortMode(t *testing.T) {
	os.Setenv("CLOUD_NAME", "CLOUD_AWS")
	utils.SetCloudName("CLOUD_AWS")
	os.Setenv("SERVICE_TYPE", "NodePort")

	// add the config and check if the sync is enabled
	AddConfigMap(t)

	if ctrl.DisableSync {
		t.Fatalf("CLOUD_AWS should be allowed in ClusterIP mode")
	}
	// validate the config
	ValidateIngress(t)
	ValidateNode(t)
	DeleteConfigMap(t)

}
