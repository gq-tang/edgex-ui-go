/*******************************************************************************
 * Copyright © 2017-2018 VMware, Inc. All Rights Reserved.
 * Copyright © 2021-2022 VMware, Inc. All Rights Reserved.
 * Copyright © 2025 IOTech Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * @author: Huaqiao Zhang, <huaqiaoz@vmware.com>
 *******************************************************************************/

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/edgexfoundry/edgex-ui-go/internal/container"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/secret"
	"github.com/edgexfoundry/go-mod-configuration/v4/configuration"
	"github.com/edgexfoundry/go-mod-configuration/v4/pkg/types"

	"github.com/gorilla/mux"
)

func (rh *ResourceHandler) DeployConfigurable(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	vars := mux.Vars(r)
	serviceKey := vars["servicekey"]
	config := make(map[string]interface{})
	var err error

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := rh.configurationCenterClient(serviceKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := client.PutConfigurationMap(config, true); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

}

func (rh *ResourceHandler) GetServiceConfig(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	vars := mux.Vars(r)
	serviceKey := vars["servicekey"]
	var err error
	client, err := rh.configurationCenterClient(serviceKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	hasConfig, err := client.HasConfiguration()
	if !hasConfig {
		http.Error(w, fmt.Sprintf("service [%s] not found on register center", serviceKey), http.StatusNotFound)
		return
	}

	config := make(map[string]interface{})

	rawConfiguration, err := client.GetConfiguration(&config)
	if err != nil {
		http.Error(w, fmt.Errorf("could not get configuration from Configuration: %v", err.Error()).Error(), http.StatusInternalServerError)
		return
	}

	actual, ok := rawConfiguration.(*map[string]interface{})
	if !ok {
		http.Error(w, "Configuration from Registry failed type check", http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(*actual)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Write(result)
}

func (rh *ResourceHandler) configurationCenterClient(serviceKey string) (configuration.Client, error) {
	config := container.ConfigurationFrom(rh.dic.Get)
	configurationConfig := types.ServiceConfig{
		Host:         config.Registry.Host,
		Port:         config.Registry.Port,
		Type:         config.Registry.Type,
		BasePath:     config.Registry.ConfigRegistryStem + config.Registry.ServiceVersion + "/" + serviceKey,
		AuthInjector: secret.NewJWTSecretProvider(bootstrapContainer.SecretProviderExtFrom(rh.dic.Get)),
	}
	client, err := configuration.NewConfigurationClient(configurationConfig)
	if err != nil {
		return nil, fmt.Errorf("connection to Registry could not be made: %v", err)
	}
	if !client.IsAlive() {
		return nil, fmt.Errorf("registry (%s) is not running", configurationConfig.Type)
	}
	return client, nil
}
