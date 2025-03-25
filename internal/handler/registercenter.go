/*******************************************************************************
 * Copyright © 2022-2023 VMware, Inc. All Rights Reserved.
 * Copyright (C) 2023 Intel Corporation
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
	"errors"
	"fmt"
	"net/http"

	"github.com/edgexfoundry/edgex-ui-go/internal/common"
	"github.com/edgexfoundry/edgex-ui-go/internal/container"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/secret"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	contractCommon "github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-registry/v4/pkg/types"
	"github.com/edgexfoundry/go-mod-registry/v4/registry"
)

func (rh *ResourceHandler) GetRegisteredServiceAll(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var err error
	var token string
	var code int
	if common.IsSecurityEnabled() {
		token, err, code = rh.getAclTokenOfConsul(w, r)
		if err != nil || code != http.StatusOK {
			http.Error(w, "unable to get consul acl token", code)
			return
		}
	}

	client, err := rh.registryCenterClient(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	endpoints, err := client.GetAllServiceEndpoints()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(endpoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Write(result)
}

func (rh *ResourceHandler) registryCenterClient(token string) (registry.Client, error) {
	bootstrapConfig := container.ConfigurationFrom(rh.dic.Get)
	registryConfig := types.Config{
		Host:            bootstrapConfig.Registry.Host,
		Port:            bootstrapConfig.Registry.Port,
		Type:            bootstrapConfig.Registry.Type,
		ServiceKey:      common.GUIServiceKey,
		ServiceHost:     bootstrapConfig.Service.Host,
		ServicePort:     bootstrapConfig.Service.Port,
		ServiceProtocol: config.DefaultHttpProtocol,
		CheckInterval:   bootstrapConfig.Service.HealthCheckInterval,
		CheckRoute:      contractCommon.ApiPingRoute,
		AuthInjector:    secret.NewJWTSecretProvider(bootstrapContainer.SecretProviderExtFrom(rh.dic.Get)),
	}
	return registry.NewRegistryClient(registryConfig)
}

func (rh *ResourceHandler) getAclTokenOfConsul(w http.ResponseWriter, r *http.Request) (string, error, int) {
	defer r.Body.Close()
	config := container.ConfigurationFrom(rh.dic.Get)
	var acl struct{ SecretID string }
	client := &http.Client{}
	sp := bootstrapContainer.SecretProviderExtFrom(rh.dic.Get)
	var url string
	if sp.IsZeroTrustEnabled() {
		url = fmt.Sprintf("http://%s:%d%s", config.Registry.Host, config.Registry.Port, AclOfConsulPathDirect)
	} else {
		url = fmt.Sprintf("http://%s:%d%s", config.APIGateway.Server, config.APIGateway.ApplicationPort, AclOfConsulPathProxied)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err, http.StatusUnauthorized
	}

	req.Header.Set(Authorization, "Bearer "+r.Header.Get("X-Consul-Token"))

	resp, err := client.Do(req)
	if err != nil {
		return "", err, http.StatusServiceUnavailable
	}

	if err := json.NewDecoder(resp.Body).Decode(&acl); err != nil {
		return "", err, http.StatusUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(""), resp.StatusCode
	}
	return acl.SecretID, nil, resp.StatusCode
}

func (rh *ResourceHandler) RegistryIsAlive(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var err error
	var token string
	var code int
	if common.IsSecurityEnabled() {
		token, err, code = rh.getAclTokenOfConsul(w, r)
		if err != nil || code != http.StatusOK {
			http.Error(w, "unable to get consul acl token", code)
			return
		}
	}

	client, err := rh.registryCenterClient(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	alive := client.IsAlive()

	if !alive {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}
