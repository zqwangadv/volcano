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

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto" // auto-registry collectors in default registry
)

const (
	// VolcanoSubSystemName - subsystem name in prometheus used by volcano
	VolcanoSubSystemName = "volcano"

	// OnSessionOpen label
	OnSessionOpen = "OnSessionOpen"

	// OnSessionClose label
	OnSessionClose = "OnSessionClose"
)

var (
	VDCUDevicesSharedNumber = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_shared_number",
			Help:      "The number of VDCU tasks sharing this card",
		},
		[]string{"devID", "NodeName"},
	)
	VDCUDevicesAllocatedMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_allocated_memory",
			Help:      "The number of VDCU memory allocated in this card",
		},
		[]string{"devID", "NodeName"},
	)
	VDCUDevicesAllocatedCores = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_allocated_cores",
			Help:      "The percentage of gpu compute cores allocated in this card",
		},
		[]string{"devID", "NodeName"},
	)
	VDCUDevicesMemoryTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_memory_limit",
			Help:      "The number of total device memory in this card",
		},
		[]string{"devID", "NodeName"},
	)
	VDCUPodMemoryAllocated = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_memory_allocation_for_a_certain_pod",
			Help:      "The VDCU device memory allocated for a certain pod",
		},
		[]string{"devID", "NodeName", "podName"},
	)
	VDCUPodCoreAllocated = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoSubSystemName,
			Name:      "vdcu_device_core_allocation_for_a_certain_pod",
			Help:      "The VDCU device core allocated for a certain pod",
		},
		[]string{"devID", "NodeName", "podName"},
	)
)

func (ds *DCUDevices) GetStatus() string {
	return ""
}

func ResetDeviceMetrics(UUID string, nodeName string, memory float64) {
	VDCUDevicesMemoryTotal.WithLabelValues(UUID, nodeName).Set(memory)
	VDCUDevicesSharedNumber.WithLabelValues(UUID, nodeName).Set(0)
	VDCUDevicesAllocatedCores.WithLabelValues(UUID, nodeName).Set(0)
	VDCUDevicesAllocatedMemory.WithLabelValues(UUID, nodeName).Set(0)

	VDCUPodMemoryAllocated.DeletePartialMatch(prometheus.Labels{"devID": UUID})
	VDCUPodCoreAllocated.DeletePartialMatch(prometheus.Labels{"devID": UUID})
}
func (ds *DCUDevices) AddPodMetrics(index int, podUID, podName string) {
	UUID := ds.Device[index].UUID
	NodeName := ds.Device[index].Node
	usage := ds.Device[index].PodMap[podUID]
	VDCUPodMemoryAllocated.WithLabelValues(UUID, NodeName, podName).Set(float64(usage.UsedMem))
	VDCUPodCoreAllocated.WithLabelValues(UUID, NodeName, podName).Set(float64(usage.UsedCore))
	VDCUDevicesSharedNumber.WithLabelValues(UUID, NodeName).Inc()
	VDCUDevicesAllocatedCores.WithLabelValues(UUID, NodeName).Set(float64(ds.Device[index].UsedCore))
	VDCUDevicesAllocatedMemory.WithLabelValues(UUID, NodeName).Set(float64(ds.Device[index].UsedMem))
}

func (ds *DCUDevices) SubPodMetrics(index int, podUID, podName string) {
	UUID := ds.Device[index].UUID
	NodeName := ds.Device[index].Node
	usage := ds.Device[index].PodMap[podUID]
	VDCUPodMemoryAllocated.WithLabelValues(UUID, NodeName, podName).Set(float64(usage.UsedMem))
	VDCUPodCoreAllocated.WithLabelValues(UUID, NodeName, podName).Set(float64(usage.UsedCore))
	if usage.UsedMem == 0 {
		delete(ds.Device[index].PodMap, podUID)
		VDCUPodMemoryAllocated.DeleteLabelValues(UUID, NodeName, podName)
		VDCUPodCoreAllocated.DeleteLabelValues(UUID, NodeName, podName)
	}
	VDCUDevicesSharedNumber.WithLabelValues(UUID, NodeName).Dec()
	VDCUDevicesAllocatedCores.WithLabelValues(UUID, NodeName).Set(float64(ds.Device[index].UsedCore))
	VDCUDevicesAllocatedMemory.WithLabelValues(UUID, NodeName).Set(float64(ds.Device[index].UsedMem))
}
