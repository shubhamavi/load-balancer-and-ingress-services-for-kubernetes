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

package k8s

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1alpha1"
	svccrd "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	gtwapiinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/nodes"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/objects"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/status"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

// Services API related functions. Parking the functions on this file instead of creating a new one since most of the functionality is same with v1alpha1pre1

func NewGtwApiInformers(cs svccrd.Interface) {
	svcApiInfomerFactory := gtwapiinformers.NewSharedInformerFactory(cs, time.Second*30)
	gwClassInformer := svcApiInfomerFactory.Networking().V1alpha1().GatewayClasses()
	gwInformer := svcApiInfomerFactory.Networking().V1alpha1().Gateways()
	lib.SetGtwAPIsInformers(&lib.GatewayAPIInformers{
		GatewayInformer:      gwInformer,
		GatewayClassInformer: gwClassInformer,
	})
}

func InformerStatusUpdatesForGtwApiGateway(key string, gateway *gatewayapi.Gateway) {
	gwStatus := gateway.Status.DeepCopy()
	defer status.UpdateGtwApiGatewayStatusObject(key, gateway, gwStatus)
	status.InitializeGtwApiGatewayConditions(gwStatus, &gateway.Spec, false)
	gwClassObj, err := lib.GetGtwAPIInformers().GatewayClassInformer.Lister().Get(gateway.Spec.GatewayClassName)
	if err != nil {
		status.UpdateGtwApiGatewayStatusGWCondition(key, gwStatus, &status.UpdateGtwApiGWStatusConditionOptions{
			Type:    string(gatewayapi.GatewayConditionScheduled),
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("Corresponding networking.x-k8s.io/gatewayclass not found %s", gateway.Spec.GatewayClassName),
			Reason:  "InvalidGatewayClass",
		})
		utils.AviLog.Warnf("key: %s, msg: Corresponding networking.x-k8s.io/gatewayclass not found %s %v",
			key, gateway.Spec.GatewayClassName, err)
		return
	}

	for _, listener := range gateway.Spec.Listeners {
		gwName, nameOk := listener.Routes.Selector.MatchLabels[lib.GtwApiGatewayNameLabelKey]
		gwNamespace, nsOk := listener.Routes.Selector.MatchLabels[lib.GtwApiGatewayNamespaceLabelKey]
		if !nameOk || !nsOk ||
			(gwName != gateway.Name) ||
			(gwNamespace != gateway.Namespace) {
			status.UpdateGtwApiGatewayStatusGWCondition(key, gwStatus, &status.UpdateGtwApiGWStatusConditionOptions{
				Type:    string(gatewayapi.GatewayConditionScheduled),
				Status:  metav1.ConditionTrue,
				Message: "Incorrect gateway matchLabels configuration",
				Reason:  "InvalidMatchLabels",
			})
			return
		}
	}

	// Additional check to see if the gatewayclass is a valid avi gateway class or not.
	if gwClassObj.Spec.Controller != lib.GtwApiAviGatewayController {
		// Return an error since this is not our object.
		status.UpdateGtwApiGatewayStatusGWCondition(key, gwStatus, &status.UpdateGtwApiGWStatusConditionOptions{
			Type:    string(gatewayapi.GatewayConditionScheduled),
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("Unable to identify controller %s", gwClassObj.Spec.Controller),
			Reason:  "UnidentifiedController",
		})
	}
}

func checkGtwApiGWForGatewayPortConflict(key string, gw *gatewayapi.Gateway) {
	found, gwSvcListeners := objects.ServiceGWLister().GetGwToSvcs(gw.Namespace + "/" + gw.Name)
	if !found {
		return
	}

	var gwProtocols []string
	// port conflicts
	for _, listener := range gw.Spec.Listeners {
		portProtoGW := string(listener.Protocol) + "/" + strconv.Itoa(int(listener.Port))
		if !utils.HasElem(gwProtocols, string(listener.Protocol)) {
			gwProtocols = append(gwProtocols, string(listener.Protocol))
		}

		if val, ok := gwSvcListeners[portProtoGW]; ok && len(val) > 1 {
			gwStatus := gw.Status.DeepCopy()
			status.UpdateGtwApiGatewayStatusListenerConditions(key, gwStatus, strconv.Itoa(int(listener.Port)), &status.UpdateGtwApiGWStatusConditionOptions{
				Type:   "PortConflict",
				Status: metav1.ConditionTrue,
				Reason: fmt.Sprintf("conflicting port configuration provided in service %s and %v", val, gwSvcListeners[portProtoGW]),
			})
			status.UpdateGtwApiGatewayStatusObject(key, gw, gwStatus)
			return
		}
	}

	// unsupported protocol
	for portProto, svcs := range gwSvcListeners {
		svcProtocol := strings.Split(portProto, "/")[0]
		if !utils.HasElem(gwProtocols, svcProtocol) {
			gwStatus := gw.Status.DeepCopy()
			status.UpdateGtwApiGatewayStatusListenerConditions(key, gwStatus, strings.Split(portProto, "/")[1], &status.UpdateGtwApiGWStatusConditionOptions{
				Type:   "UnsupportedProtocol",
				Status: metav1.ConditionTrue,
				Reason: fmt.Sprintf("Unsupported protocol found in services %v", svcs),
			})
			status.UpdateGtwApiGatewayStatusObject(key, gw, gwStatus)
			return
		}
	}
}

func checkSvcForGtwApiGatewayPortConflict(svc *corev1.Service, key string) {
	gateway, portProtocols := nodes.ParseL4ServiceForGateway(svc, key)
	if gateway == "" {
		utils.AviLog.Warnf("key: %s, msg: Unable to find gateway labels in service", key)
		return
	}

	found, gwSvcListeners := objects.ServiceGWLister().GetGwToSvcs(gateway)
	if !found {
		return
	}

	gwNSName := strings.Split(gateway, "/")
	gw, err := lib.GetGtwAPIInformers().GatewayInformer.Lister().Gateways(gwNSName[0]).Get(gwNSName[1])
	if err != nil {
		utils.AviLog.Warnf("key: %s, msg: Unable to find gateway: %v", key, err)
		return
	}

	// detect port conflict
	for _, portProtocol := range portProtocols {
		if val, ok := gwSvcListeners[portProtocol]; ok {
			if !utils.HasElem(val, svc.Namespace+"/"+svc.Name) {
				val = append(val, svc.Namespace+"/"+svc.Name)
			}
			if len(val) > 1 {
				portProtocolArr := strings.Split(portProtocol, "/")
				gwStatus := gw.Status.DeepCopy()
				status.UpdateGtwApiGatewayStatusListenerConditions(key, gwStatus, portProtocolArr[1], &status.UpdateGtwApiGWStatusConditionOptions{
					Type:   "PortConflict",
					Status: metav1.ConditionTrue,
					Reason: fmt.Sprintf("conflicting port configuration provided in service %s and %s/%s", val, svc.Namespace, svc.Name),
				})
				status.UpdateGtwApiGatewayStatusObject(key, gw, gwStatus)
				return
			}
		}
	}

	// detect unsupported protocol
	// TODO

	return
}

// SetupGatewayAPI handles setting up of GatewayAPI event handlers
func (c *AviController) SetupGtwApiEventHandlers(numWorkers uint32) {
	utils.AviLog.Infof("Setting up GatewayAPI Event handlers")
	informer := lib.GetGtwAPIInformers()

	gatewayEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if c.DisableSync {
				return
			}
			gw := obj.(*gatewayapi.Gateway)
			namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gw))
			if !utils.CheckIfNamespaceAccepted(namespace) {
				utils.AviLog.Debugf("Gateway add event. Namespace %s didn't qualify filter. Not adding gateway.", namespace)
				return
			}
			key := lib.Gateway + "/" + utils.ObjKey(gw)
			utils.AviLog.Infof("key: %s, msg: ADD", key)

			InformerStatusUpdatesForGtwApiGateway(key, gw)
			checkGtwApiGWForGatewayPortConflict(key, gw)

			bkt := utils.Bkt(namespace, numWorkers)
			c.workqueue[bkt].AddRateLimited(key)
		},
		UpdateFunc: func(old, new interface{}) {
			if c.DisableSync {
				return
			}
			oldObj := old.(*gatewayapi.Gateway)
			gw := new.(*gatewayapi.Gateway)

			if !reflect.DeepEqual(oldObj.Spec, gw.Spec) || gw.GetDeletionTimestamp() != nil {
				namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gw))
				if !utils.CheckIfNamespaceAccepted(namespace) {
					utils.AviLog.Debugf("Gateway update event. Namespace %s didn't qualify filter. Not updating gateway.", namespace)
					return
				}
				key := lib.Gateway + "/" + utils.ObjKey(gw)
				utils.AviLog.Infof("key: %s, msg: UPDATE", key)

				InformerStatusUpdatesForGtwApiGateway(key, gw)
				checkGtwApiGWForGatewayPortConflict(key, gw)

				bkt := utils.Bkt(namespace, numWorkers)
				c.workqueue[bkt].AddRateLimited(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if c.DisableSync {
				return
			}
			gw := obj.(*gatewayapi.Gateway)
			namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gw))
			if !utils.CheckIfNamespaceAccepted(namespace) {
				utils.AviLog.Debugf("Gateway delete event. Namespace %s didn't qualify filter. Not deleting gateway.", namespace)
				return
			}
			key := lib.Gateway + "/" + utils.ObjKey(gw)
			utils.AviLog.Infof("key: %s, msg: DELETE", key)
			bkt := utils.Bkt(namespace, numWorkers)
			c.workqueue[bkt].AddRateLimited(key)
		},
	}

	gatewayClassEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if c.DisableSync {
				return
			}
			gwclass := obj.(*gatewayapi.GatewayClass)
			namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gwclass))
			key := lib.GatewayClass + "/" + utils.ObjKey(gwclass)
			utils.AviLog.Infof("key: %s, msg: ADD", key)
			bkt := utils.Bkt(namespace, numWorkers)
			c.workqueue[bkt].AddRateLimited(key)
		},
		UpdateFunc: func(old, new interface{}) {
			if c.DisableSync {
				return
			}
			oldObj := old.(*gatewayapi.GatewayClass)
			gwclass := new.(*gatewayapi.GatewayClass)
			if !reflect.DeepEqual(oldObj.Spec, gwclass.Spec) {
				namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gwclass))
				key := lib.GatewayClass + "/" + utils.ObjKey(gwclass)
				utils.AviLog.Infof("key: %s, msg: UPDATE", key)
				bkt := utils.Bkt(namespace, numWorkers)
				c.workqueue[bkt].AddRateLimited(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if c.DisableSync {
				return
			}
			gwclass := obj.(*gatewayapi.GatewayClass)
			key := lib.GatewayClass + "/" + utils.ObjKey(gwclass)
			namespace, _, _ := cache.SplitMetaNamespaceKey(utils.ObjKey(gwclass))
			utils.AviLog.Infof("key: %s, msg: DELETE", key)
			bkt := utils.Bkt(namespace, numWorkers)
			c.workqueue[bkt].AddRateLimited(key)
		},
	}

	informer.GatewayInformer.Informer().AddEventHandler(gatewayEventHandler)
	informer.GatewayInformer.Informer().AddIndexers(
		cache.Indexers{
			lib.GatewayClassGatewayIndex: func(obj interface{}) ([]string, error) {
				gw, ok := obj.(*gatewayapi.Gateway)
				if !ok {
					return []string{}, nil
				}
				return []string{gw.Spec.GatewayClassName}, nil
			},
		},
	)

	informer.GatewayClassInformer.Informer().AddEventHandler(gatewayClassEventHandler)
	informer.GatewayClassInformer.Informer().AddIndexers(
		cache.Indexers{
			lib.AviSettingGWClassIndex: func(obj interface{}) ([]string, error) {
				gwclass, ok := obj.(*gatewayapi.GatewayClass)
				if !ok {
					return []string{}, nil
				}
				if gwclass.Spec.ParametersRef != nil {
					// sample settingKey: ako.vmware.com/AviInfraSetting/avi-1
					settingKey := gwclass.Spec.ParametersRef.Group + "/" + gwclass.Spec.ParametersRef.Kind + "/" + gwclass.Spec.ParametersRef.Name
					return []string{settingKey}, nil
				}
				return []string{}, nil
			},
		},
	)

	return
}
