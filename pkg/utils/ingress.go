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

package utils

import (
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var IngressApiMap = map[string]string{
	"corev1":  NetV1IngressInformer,
	"v1beta1": NetV1beta1IngressInformer,
}

var (
	NetworkingV1beta1Ingress = schema.GroupVersionResource{
		Group:    "networkingv1.k8s.io",
		Version:  "v1beta1",
		Resource: "ingresses",
	}

	NetworkingIngress = schema.GroupVersionResource{
		Group:    "networkingv1.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}
)

// func GetIngressApi(kc kubernetes.Interface) string {
// 	ingressAPI := os.Getenv("INGRESS_API")
// 	if ingressAPI != "" {
// 		ingressAPI, ok := IngressApiMap[ingressAPI]
// 		if !ok {
// 			return NetV1IngressInformer
// 		}
// 		return ingressAPI
// 	}

// 	var timeout int64
// 	timeout = 120

// 	_, ingErr := kc.NetworkingV1().Ingresses("").List(metav1.ListOptions{TimeoutSeconds: &timeout})
// 	// _, ingErr := kc.NetworkingV1().Ingresses("").List(metav1.ListOptions{TimeoutSeconds: &timeout})
// 	if ingErr != nil {
// 		AviLog.Infof("networking/v1 ingresses not found, setting informer for networking/v1beta1: %v", ingErr)
// 		return NetV1beta1IngressInformer
// 	}
// 	return NetV1IngressInformer
// }

func fromBeta(old *networkingv1beta1.Ingress) (*networkingv1.Ingress, error) {
	networkingIngress := &networkingv1.Ingress{}

	err := runtimeScheme.Convert(old, networkingIngress, nil)
	if err != nil {
		return nil, err
	}

	return networkingIngress, nil
}

func fromGA(old *networkingv1.Ingress) (*networkingv1beta1.Ingress, error) {
	networkingv1beta1sIngress := &networkingv1beta1.Ingress{}

	err := runtimeScheme.Convert(old, networkingv1beta1sIngress, nil)
	if err != nil {
		return nil, err
	}

	return networkingv1beta1sIngress, nil
}

// ToNetworkingV1Ingress converts obj interface to networkingv1.Ingress
func ToNetworkingV1Ingress(obj interface{}) (*networkingv1.Ingress, bool) {
	oldVersion, inBeta := obj.(*networkingv1beta1.Ingress)
	if inBeta {
		ing, err := fromBeta(oldVersion)
		if err != nil {
			AviLog.Warnf("unexpected error converting Ingress from networking/v1beta1 package: %v", err)
			return nil, false
		}

		return ing, true
	}

	if ing, ok := obj.(*networkingv1.Ingress); ok {
		return ing, true
	}

	return nil, false
}

// ToExtensionIngress converts obj interface to networkingv1beta1.Ingress
func ToExtensionIngress(obj interface{}) (*networkingv1beta1.Ingress, bool) {
	oldVersion, inExtension := obj.(*networkingv1.Ingress)
	if inExtension {
		ing, err := fromGA(oldVersion)
		if err != nil {
			AviLog.Warnf("unexpected error converting Ingress from networking package: %v", err)
			return nil, false
		}

		return ing, true
	}

	if ing, ok := obj.(*networkingv1beta1.Ingress); ok {
		return ing, true
	}

	return nil, false
}
