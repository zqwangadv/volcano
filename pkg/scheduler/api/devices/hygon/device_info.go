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
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
	"time"

	"volcano.sh/volcano/pkg/scheduler/api/devices"
	deviceconfig "volcano.sh/volcano/pkg/scheduler/api/devices/config"
	"volcano.sh/volcano/pkg/scheduler/plugins/util/nodelock"
)

type DCUUsage struct {
	UsedMem  uint
	UsedCore uint
}

// DCUDevice include DCU id, memory and the pods that are sharing it.
type DCUDevice struct {
	// GPU ID
	ID int
	// Node this DCU Device belongs
	Node string
	// DCU Unique ID
	UUID string
	// The resource usage by pods that are sharing this DCU
	PodMap map[string]*DCUUsage
	// memory per card
	Memory uint
	// max sharing number
	Number uint
	// type of this number
	Type string
	// Health condition of this DCU
	Health bool
	// number of allocated
	UsedNum uint
	// number of device memory allocated
	UsedMem uint
	// number of core used
	UsedCore uint
}

type DCUDevices struct {
	Name string
	// We cache score in filter step according to schedulePolicy, to avoid recalculating in score
	Score float64

	Device map[int]*DCUDevice
}

func NewDCUDevices(name string, node *v1.Node) *DCUDevices {
	if node == nil {
		return nil
	}
	annos, ok := node.Annotations[deviceconfig.HygonVDCURegister]
	if !ok {
		return nil
	}
	handshake, ok := node.Annotations[deviceconfig.HygonVDCUHandshake]
	if !ok {
		return nil
	}
	nodedevices := decodeNodeDevices(name, annos)
	if (nodedevices == nil) || len(nodedevices.Device) == 0 {
		return nil
	}

	for _, val := range nodedevices.Device {
		klog.V(3).InfoS("Hygon Device registered name", "name", nodedevices.Name, "val", *val)
		ResetDeviceMetrics(val.UUID, node.Name, float64(val.Memory))
	}

	// We have to handshake here in order to avoid time-inconsistency between scheduler and nodes
	if strings.Contains(handshake, "Requesting") {
		formertime, _ := time.Parse("2006.01.02 15:04:05", strings.Split(handshake, "_")[1])
		if time.Now().After(formertime.Add(time.Second * 60)) {
			klog.V(3).Infof("node %v device %s leave", node.Name, handshake)

			tmppat := make(map[string]string)
			tmppat[deviceconfig.HygonVDCUHandshake] = "Deleted_" + time.Now().Format("2006.01.02 15:04:05")
			patchNodeAnnotations(node, tmppat)
			return nil
		}
	} else if strings.Contains(handshake, "Deleted") {
		return nil
	} else {
		tmppat := make(map[string]string)
		tmppat[deviceconfig.HygonVDCUHandshake] = "Requesting_" + time.Now().Format("2006.01.02 15:04:05")
		patchNodeAnnotations(node, tmppat)
	}
	return nodedevices
}

func (ds *DCUDevices) ScoreNode(pod *v1.Pod, schedulePolicy string) float64 {
	/* TODO: we need a base score to be campatable with preemption, it means a node without evicting a task has
	   a higher score than those needs to evict a task */

	// Use cached stored in filter state in order to avoid recalculating.
	return ds.Score
}

func (ds *DCUDevices) GetIgnoredDevices() []string {
	return []string{}
}

func (ds *DCUDevices) AddQueueResource(pod *v1.Pod) map[string]float64 {
	if ds == nil {
		return map[string]float64{}
	}
	klog.V(5).InfoS("AddQueueResource", "Name", pod.Name)
	res := map[string]float64{}
	ids, ok := pod.Annotations[AssignedIDsAllocatedAnnotations]
	if !ok {
		klog.Errorf("pod %s has no annotation hami.io/dcu-devices-allocated", pod.Name)
		return res
	}
	podDev := decodePodDevices(ids)
	for _, val := range podDev {
		for _, deviceused := range val {
			for _, gsdevice := range ds.Device {
				if strings.Contains(deviceused.UUID, gsdevice.UUID) {
					res[getConfig().ResourceMemoryName] += float64(deviceused.Usedmem * 1000)
					res[getConfig().ResourceCoreName] += float64(deviceused.Usedcores * 1000)
				}
			}
		}
	}
	klog.V(4).InfoS("AddQueueResource", "Name=", pod.Name, "res=", res)
	return res
}

// AddResource adds the pod to GPU pool if it is assigned
func (ds *DCUDevices) AddResource(pod *v1.Pod) {
	if ds == nil {
		return
	}

	ds.addResource(pod.Annotations, pod)
}

func (ds *DCUDevices) addResource(annotations map[string]string, pod *v1.Pod) {
	ids, ok := annotations[AssignedIDsAllocatedAnnotations]
	if !ok {
		klog.Errorf("pod %s has no annotation hami.io/dcu-devices-allocated", pod.Name)
		return
	}
	podDev := decodePodDevices(ids)
	for _, val := range podDev {
		for _, deviceused := range val {
			for index, gsdevice := range ds.Device {
				if strings.Contains(deviceused.UUID, gsdevice.UUID) {
					ds.addToPodMap(annotations, pod)
					ds.AddPodMetrics(index, string(pod.UID), pod.Name)
				}
			}
		}
	}
}

func (ds *DCUDevices) addToPodMap(annotations map[string]string, pod *v1.Pod) {
	ids, ok := annotations[AssignedIDsAllocatedAnnotations]
	if !ok {
		klog.Errorf("pod %s has no annotation hami.io/dcu-devices-allocated", pod.Name)
		return
	}
	podDev := decodePodDevices(ids)
	for _, val := range podDev {
		for _, deviceused := range val {
			for idx, gsdevice := range ds.Device {
				if strings.Contains(deviceused.UUID, gsdevice.UUID) {
					podUID := string(pod.UID)
					_, ok := gsdevice.PodMap[podUID]
					if !ok {
						ds.Device[idx].PodMap[podUID] = &DCUUsage{
							UsedMem:  0,
							UsedCore: 0,
						}
					}
					ds.Device[idx].UsedMem += deviceused.Usedmem
					ds.Device[idx].UsedCore += deviceused.Usedcores
					ds.Device[idx].PodMap[podUID].UsedMem += deviceused.Usedmem
					ds.Device[idx].PodMap[podUID].UsedCore += deviceused.Usedcores
				}
			}
		}
	}
}

// SubResource frees the gpu hold by the pod
func (ds *DCUDevices) SubResource(pod *v1.Pod) {
	if ds == nil {
		return
	}
	ids, ok := pod.Annotations[AssignedIDsAllocatedAnnotations]
	if !ok {
		return
	}
	podDev := decodePodDevices(ids)
	for _, val := range podDev {
		for _, deviceused := range val {
			for index, gsdevice := range ds.Device {
				if strings.Contains(deviceused.UUID, gsdevice.UUID) {
					ds.SubPodMetrics(index, string(pod.UID), pod.Name)
				}
			}
		}
	}
}

func (ds *DCUDevices) HasDeviceRequest(pod *v1.Pod) bool {
	if HygonHAMiDCUEnable && checkVDCUResourcesInPod(pod) {
		return true
	}
	return false
}

func (ds *DCUDevices) Release(kubeClient kubernetes.Interface, pod *v1.Pod) error {
	return nil
}

func (ds *DCUDevices) FilterNode(pod *v1.Pod, schedulePolicy string) (int, string, error) {
	if HygonHAMiDCUEnable {
		klog.V(4).Infoln("hami-vdcu DeviceSharing starts filtering pods", pod.Name)
		fit, _, score, err := checkNodeDCUSharingPredicateAndScore(pod, ds, true, schedulePolicy)
		if err != nil || !fit {
			klog.ErrorS(err, "Failed to fitler node to vdcu task", "pod", pod.Name)
			return devices.Unschedulable, "hami-vdcu DeviceSharing error", err
		}
		ds.Score = score
		klog.V(4).Infoln("hami-vdcu DeviceSharing successfully filters pods")
	}
	return devices.Success, "", nil
}

func (ds *DCUDevices) Allocate(kubeClient kubernetes.Interface, pod *v1.Pod) error {
	if HygonHAMiDCUEnable {
		klog.V(4).Infoln("hami-vdcu DeviceSharing:Into AllocateToPod", pod.Name)
		fit, device, _, err := checkNodeDCUSharingPredicateAndScore(pod, ds, false, "")
		if err != nil || !fit {
			klog.ErrorS(err, "Failed to allocate vdcu task", "pod", pod.Name)
			return err
		}
		if NodeLockEnable {
			nodelock.UseClient(kubeClient)
			err = nodelock.LockNode(ds.Name, DeviceName)
			if err != nil {
				return errors.Errorf("node %s locked for %s hami-vdcu lockname %s", ds.Name, pod.Name, err.Error())
			}
		}

		annotations := make(map[string]string)
		annotations[AssignedNodeAnnotations] = ds.Name
		annotations[AssignedTimeAnnotations] = strconv.FormatInt(time.Now().Unix(), 10)
		podDevicesAnnotations := encodePodDevices(device)
		annotations[AssignedIDsToAllocateAnnotations] = podDevicesAnnotations
		annotations[AssignedIDsAllocatedAnnotations] = podDevicesAnnotations

		annotations[DeviceBindPhase] = "allocating"
		annotations[BindTimeAnnotations] = strconv.FormatInt(time.Now().Unix(), 10)
		// To avoid that the pod allocated info updating latency, add it first
		ds.addToPodMap(annotations, pod)
		err = patchPodAnnotations(kubeClient, pod, annotations)
		if err != nil {
			return err
		}

		klog.V(3).Infoln("DeviceSharing:Allocate Success")
	}
	return nil
}

func (ds *DCUDevices) TryAddPod(device *DCUDevice, card uint, u uint) (bool, string) {
	device.UsedNum++
	device.UsedMem += card
	device.UsedCore += u
	return true, device.UUID
}
