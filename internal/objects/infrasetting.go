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

package objects

import (
	"sync"
)

var infral7lister *AviInfraSettingL7Lister
var infraonce sync.Once

func InfraSettingL7Lister() *AviInfraSettingL7Lister {
	infraonce.Do(func() {
		infral7lister = &AviInfraSettingL7Lister{
			IngRouteInfraSettingStore: NewObjectMapStore(),
		}
	})
	return infral7lister
}

type AviInfraSettingL7Lister struct {
	InfraSettingIngRouteLock sync.RWMutex

	// namespaced ingress/route -> infrasetting
	IngRouteInfraSettingStore *ObjectMapStore
}

func (v *AviInfraSettingL7Lister) GetIngRouteToInfraSetting(ingrouteNsName string) (bool, string) {
	found, infraSetting := v.IngRouteInfraSettingStore.Get(ingrouteNsName)
	if !found {
		return false, ""
	}
	return true, infraSetting.(string)
}

func (v *AviInfraSettingL7Lister) UpdateIngRouteInfraSettingMappings(infraSetting, ingrouteNsName string) {
	v.InfraSettingIngRouteLock.Lock()
	defer v.InfraSettingIngRouteLock.Unlock()
	v.IngRouteInfraSettingStore.AddOrUpdate(ingrouteNsName, infraSetting)
}

func (v *AviInfraSettingL7Lister) RemoveIngRouteInfraSettingMappings(ingrouteNsName string) bool {
	v.InfraSettingIngRouteLock.Lock()
	defer v.InfraSettingIngRouteLock.Unlock()
	return v.IngRouteInfraSettingStore.Delete(ingrouteNsName)
}
