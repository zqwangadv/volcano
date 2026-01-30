/*
Copyright 2026 The Volcano Authors.

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

package vdcu

const (
	// DeviceName used to indicate this device
	DeviceName = "hamivdcu"

	DCUInUseType                     = "hygon.com/use-dcutype"
	DCUNoUseType                     = "hygon.com/nouse-dcutype"
	DCUInUseUUID                     = "hygon.com/use-gpuuuid"
	DCUNoUseUUID                     = "hygon.com/nouse-gpuuuid"
	AssignedTimeAnnotations          = "hami.io/vgpu-time"
	AssignedIDsToAllocateAnnotations = "hami.io/dcu-devices-to-allocate"
	AssignedIDsAllocatedAnnotations  = "hami.io/dcu-devices-allocated"
	AssignedNodeAnnotations          = "hami.io/vgpu-node"
	BindTimeAnnotations              = "hami.io/bind-time"
	DeviceBindPhase                  = "hami.io/bind-phase"

	HygonDCUDevice = "DCU"

	// binpack means the lower device memory remained after this allocation, the better
	binpackPolicy = "binpack"
	// spread means better put this task into an idle GPU card than a shared GPU card
	spreadPolicy = "spread"

	binpackMultiplier = 100
	spreadMultiplier  = 100
)

var (
	HygonHAMiDCUEnable bool
	NodeLockEnable     bool
)

type ContainerDevice struct {
	UUID string
	// device type, like NVIDIA, MLU
	Type      string
	Usedmem   uint
	Usedcores uint
}

type ContainerDevices []ContainerDevice
