/*
 * Copyright 2020-2021 VMware, Inc.
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

package lib

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gtwapiv1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
	gtwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	svcInformer "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions/apis/v1alpha1"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

var gtwAPICS gtwapi.Interface
var gtwAPIInformers *GatewayAPIInformers

func SetGatewayAPIClientset(cs gtwapi.Interface) {
	gtwAPICS = cs
}

func GetGatewayAPIClientset() gtwapi.Interface {
	return gtwAPICS
}

type GatewayAPIInformers struct {
	GatewayInformer      svcInformer.GatewayInformer
	GatewayClassInformer svcInformer.GatewayClassInformer
}

func SetGtwAPIsInformers(c *GatewayAPIInformers) {
	gtwAPIInformers = c
}

func GetGtwAPIInformers() *GatewayAPIInformers {
	return gtwAPIInformers
}

func RemoveGtwApiGatewayFinalizer(gw *gtwapiv1alpha1.Gateway) {
	finalizers := utils.Remove(gw.GetFinalizers(), GatewayFinalizer)
	gw.SetFinalizers(finalizers)
	UpdateGtwApiGatewayFinalizer(gw)
}

func CheckAndSetGtwApiGatewayFinalizer(gw *gtwapiv1alpha1.Gateway) {
	if !ContainsFinalizer(gw, GatewayFinalizer) {
		finalizers := append(gw.GetFinalizers(), GatewayFinalizer)
		gw.SetFinalizers(finalizers)
		UpdateGtwApiGatewayFinalizer(gw)
	}
}

func UpdateGtwApiGatewayFinalizer(gw *gtwapiv1alpha1.Gateway) {
	patchPayload, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string][]string{
			"finalizers": gw.GetFinalizers(),
		},
	})

	_, err := GetGatewayAPIClientset().NetworkingV1alpha1().Gateways(gw.Namespace).Patch(context.TODO(), gw.Name, types.MergePatchType, patchPayload, metav1.PatchOptions{})
	if err != nil {
		utils.AviLog.Warnf("error while patching the gateway with updated finalizers, %v", err)
		return
	}

	utils.AviLog.Infof("Successfully patched the gateway with finalizers: %v", gw.GetFinalizers())
}
