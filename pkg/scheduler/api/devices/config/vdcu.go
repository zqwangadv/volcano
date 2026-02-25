/*
Copyright 2026 The Hygon Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

const (
	// HygonVDCUMemory extended gpu memory
	HygonVDCUMemory = "hygon.com/dcumem"
	// HygonVDCUCores indicates utilization percentage of VDCU
	HygonVDCUCores = "hygon.com/dcucores"
	// HygonVDCUNumber virtual GPU card number
	HygonVDCUNumber = "hygon.com/dcunum"
	// HygonVDCURegister virtual gpu information registered from device-plugin to scheduler
	HygonVDCURegister = "hami.io/node-dcu-register"
	// HygonVDCUHandshake for VDCU
	HygonVDCUHandshake = "hami.io/node-handshake-dcu"
)

// HygonConfig is used for Hygon-VDCU
type HygonConfig struct {
	// ResourceCountName is the name of GPU count
	ResourceCountName string `yaml:"resourceCountName"`
	// ResourceMemoryName is the name of GPU device memory
	ResourceMemoryName string `yaml:"resourceMemoryName"`
	// ResourceCoreName is the name of GPU core
	ResourceCoreName string `yaml:"resourceCoreName"`
	// ResourceMemoryPercentageName is the name of GPU device memory
	ResourceMemoryPercentageName string `yaml:"resourceMemoryPercentageName"`
	// ResourcePriority is the name of GPU priority
	ResourcePriority string `yaml:"resourcePriorityName"`
	// OverwriteEnv is whether we overwrite 'NVIDIA_VISIBLE_DEVICES' to 'none' for non-gpu tasks
	OverwriteEnv bool `yaml:"overwriteEnv"`
	// DefaultMemory is the number of device memory if not specified
	DefaultMemory uint `yaml:"defaultMemory"`
	// DefaultCores is the number of device cores if not specified
	DefaultCores uint `yaml:"defaultCores"`
	// DefaultGPUNum is the number of device number if not specified
	DefaultDCUNum uint `yaml:"defaultGPUNum"`
	// DeviceSplitCount is the number of fake-devices reported by VDCU-device-plugin per GPU
	DeviceSplitCount uint `yaml:"deviceSplitCount"`
	// DeviceMemoryScaling is the device memory oversubscription factor
	DeviceMemoryScaling float64 `yaml:"deviceMemoryScaling"`
	// DeviceCoreScaling is the device core oversubscription factor
	DeviceCoreScaling float64 `yaml:"deviceCoreScaling"`
}
