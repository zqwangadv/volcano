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
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"volcano.sh/volcano/pkg/scheduler/api/devices"
	"volcano.sh/volcano/pkg/scheduler/api/devices/config"
)

func decodeNodeDevices(name, str string) *DCUDevices {
	if !strings.Contains(str, ":") {
		return nil
	}
	tmp := strings.Split(str, ":")
	retval := &DCUDevices{
		Name:   name,
		Device: make(map[int]*DCUDevice),
		Score:  float64(0),
	}
	for index, val := range tmp {
		if strings.Contains(val, ",") {
			items := strings.Split(val, ",")
			if len(items) < 6 {
				klog.Error("wrong Node DCU info: ", val)
				return nil
			}
			count, _ := strconv.Atoi(items[1])
			devmem, _ := strconv.Atoi(items[2])
			health, _ := strconv.ParseBool(items[6])
			i := DCUDevice{
				ID:     index,
				Node:   name,
				UUID:   items[0],
				Number: uint(count),
				Memory: uint(devmem),
				Type:   items[4],
				PodMap: make(map[string]*DCUUsage),
				Health: health,
			}

			retval.Device[index] = &i
		}
	}
	return retval
}

func encodeContainerDevices(cd []ContainerDevice) string {
	tmp := ""
	for _, val := range cd {
		tmp += val.UUID + "," + val.Type + "," + strconv.Itoa(int(val.Usedmem)) + "," + strconv.Itoa(int(val.Usedcores)) + ":"
	}
	klog.V(4).Infoln("Encoded container Devices=", tmp)
	return tmp
	//return strings.Join(cd, ",")
}

func encodePodDevices(pd []ContainerDevices) string {
	var ss []string
	for _, cd := range pd {
		ss = append(ss, encodeContainerDevices(cd))
	}
	return strings.Join(ss, ";")
}

func decodeContainerDevices(str string) ContainerDevices {
	if len(str) == 0 {
		return ContainerDevices{}
	}
	cd := strings.Split(str, ":")
	contdev := ContainerDevices{}
	tmpdev := ContainerDevice{}
	if len(str) == 0 {
		return contdev
	}
	for _, val := range cd {
		if strings.Contains(val, ",") {
			tmpstr := strings.Split(val, ",")
			tmpdev.UUID = tmpstr[0]
			tmpdev.Type = tmpstr[1]
			devmem, _ := strconv.ParseInt(tmpstr[2], 10, 32)
			tmpdev.Usedmem = uint(devmem)
			devcores, _ := strconv.ParseInt(tmpstr[3], 10, 32)
			tmpdev.Usedcores = uint(devcores)
			contdev = append(contdev, tmpdev)
		}
	}
	return contdev
}

func decodePodDevices(str string) []ContainerDevices {
	if len(str) == 0 {
		return []ContainerDevices{}
	}
	var pd []ContainerDevices
	for _, s := range strings.Split(str, ";") {
		cd := decodeContainerDevices(s)
		pd = append(pd, cd)
	}
	return pd
}

func checkVDCUResourcesInPod(pod *v1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		_, ok := container.Resources.Limits[v1.ResourceName(getConfig().ResourceMemoryName)]
		if ok {
			return true
		}
		_, ok = container.Resources.Limits[v1.ResourceName(getConfig().ResourceCountName)]
		if ok {
			return true
		}
	}
	return false
}

func resourcereqs(pod *v1.Pod) []devices.ContainerDeviceRequest {
	countName := getConfig().ResourceCountName
	memoryName := getConfig().ResourceMemoryName
	percentageName := getConfig().ResourceMemoryPercentageName
	coreName := getConfig().ResourceCoreName
	return devices.ExtractResourceRequest(pod, "DCU", countName, memoryName, percentageName, coreName)
}

func checkDCUtype(annos map[string]string, cardtype string) bool {
	inuse, ok := annos[DCUInUseType]
	if ok {
		if !strings.Contains(inuse, ",") {
			if strings.Contains(strings.ToUpper(cardtype), strings.ToUpper(inuse)) {
				return true
			}
		} else {
			for _, val := range strings.Split(inuse, ",") {
				if strings.Contains(strings.ToUpper(cardtype), strings.ToUpper(val)) {
					return true
				}
			}
		}
		return false
	}
	nouse, ok := annos[DCUNoUseType]
	if ok {
		if !strings.Contains(nouse, ",") {
			if strings.Contains(strings.ToUpper(cardtype), strings.ToUpper(nouse)) {
				return false
			}
		} else {
			for _, val := range strings.Split(nouse, ",") {
				if strings.Contains(strings.ToUpper(cardtype), strings.ToUpper(val)) {
					return false
				}
			}
		}
		return true
	}
	return true
}

func checkDCUUUID(annos map[string]string, cardUUID string) bool {
	inuse, ok := annos[DCUInUseUUID]
	if ok {
		if !strings.Contains(inuse, ",") {
			if strings.Contains(strings.ToUpper(cardUUID), strings.ToUpper(inuse)) {
				return true
			}
		} else {
			for _, val := range strings.Split(inuse, ",") {
				if strings.Contains(strings.ToUpper(cardUUID), strings.ToUpper(val)) {
					return true
				}
			}
		}
		return false
	}
	nouse, ok := annos[DCUNoUseUUID]
	if ok {
		if !strings.Contains(nouse, ",") {
			if strings.Contains(strings.ToUpper(cardUUID), strings.ToUpper(nouse)) {
				return false
			}
		} else {
			for _, val := range strings.Split(nouse, ",") {
				if strings.Contains(strings.ToUpper(cardUUID), strings.ToUpper(val)) {
					return false
				}
			}
		}
		return true
	}
	return true
}

func checkType(annos map[string]string, d DCUDevice, n devices.ContainerDeviceRequest) bool {
	//General type check, NVIDIA->NVIDIA MLU->MLU
	if !strings.Contains(d.Type, n.Type) {
		return false
	}
	if n.Type == HygonDCUDevice {
		return checkDCUtype(annos, d.Type)
	}
	klog.Errorf("Unrecognized device %v", n.Type)
	return false
}

// getGPUDeviceSnapShot is not a strict deep copy, the pointer item is same with origin.
func getDCUDeviceSnapShot(snap *DCUDevices) *DCUDevices {
	ret := DCUDevices{
		Name:   snap.Name,
		Device: make(map[int]*DCUDevice),
		Score:  float64(0),
	}
	for index, val := range snap.Device {
		if val != nil {
			ret.Device[index] = &DCUDevice{
				ID:       val.ID,
				Node:     val.Node,
				UUID:     val.UUID,
				PodMap:   val.PodMap,
				Memory:   val.Memory,
				Number:   val.Number,
				Type:     val.Type,
				Health:   val.Health,
				UsedNum:  val.UsedNum,
				UsedMem:  val.UsedMem,
				UsedCore: val.UsedCore,
			}
			klog.V(4).Infoln("getDCUDeviceSnapShot:", ret.Device[index].UsedMem, val.UsedMem, ret.Device[index].UsedCore, val.UsedCore)
		}
	}
	return &ret
}

// checkNodeDCUSharingPredicate checks if a pod with vdcu requirement can be scheduled on a node.
func checkNodeDCUSharingPredicateAndScore(pod *v1.Pod, dssnap *DCUDevices, replicate bool, schedulePolicy string) (bool, []ContainerDevices, float64, error) {
	// no gpu sharing request
	score := float64(0)
	if !checkVDCUResourcesInPod(pod) {
		return true, []ContainerDevices{}, 0, nil
	}

	ctrReq := resourcereqs(pod)
	if len(ctrReq) == 0 {
		return true, []ContainerDevices{}, 0, nil
	}

	var ds *DCUDevices
	if replicate {
		ds = getDCUDeviceSnapShot(dssnap)
	} else {
		ds = dssnap
	}

	// Select best card for exclusive request
	topologyInfo, err := GetTopologyInfo(dssnap.Name)
	if err != nil && topologyInfo == nil {
		klog.V(3).Infof("No DCU topology configmap found %v", err)
	}

	ctrdevs := []ContainerDevices{}
	for _, val := range ctrReq {
		devs := []ContainerDevice{}
		var allocatedIdx, preAllocatedIdx []int
		var filterTopologyInfo *Topology

		if int(val.Nums) > len(ds.Device) {
			return false, []ContainerDevices{}, 0, fmt.Errorf("no enough dcu cards on node %s", ds.Name)
		}
		klog.V(3).InfoS("Allocating device for container", "request", val)

		if _, ok := pod.Annotations[DCUInUseUUID]; ok {
			goto loop
		}

		// Deal with exclusive request
		if val.MemPercentagereq == 100 && val.Coresreq == 100 && topologyInfo != nil {
			for i := len(ds.Device) - 1; i >= 0; i-- {
				if ds.Device[i].UsedCore > 0 || ds.Device[i].UsedMem > 0 {
					allocatedIdx = append(allocatedIdx, ds.Device[i].ID)
				}
			}
			filterTopologyInfo = FilterTopologyByAllocated(topologyInfo, allocatedIdx)
			preAllocatedIdx = SelectBestDevices(filterTopologyInfo, int(val.Nums))
			if len(preAllocatedIdx) == 0 {
				return false, []ContainerDevices{}, 0, fmt.Errorf("no enough dcu cards on node %s for topology select", ds.Name)
			}

			for _, i := range preAllocatedIdx {
				if !checkType(pod.Annotations, *ds.Device[i], val) {
					klog.Errorln("failed checktype", ds.Device[i].Type, val.Type)
					return false, []ContainerDevices{}, 0, fmt.Errorf("failed checktype", ds.Device[i].Type, val.Type)
				}

				_, uuid := ds.TryAddPod(ds.Device[i], uint(float64(ds.Device[i].Memory)*float64(val.MemPercentagereq)/100.0), uint(val.Coresreq))
				klog.V(3).Info("fitted uuid: ", uuid)
				devs = append(devs, ContainerDevice{
					UUID:      uuid,
					Type:      val.Type,
					Usedmem:   uint(float64(ds.Device[i].Memory) * float64(val.MemPercentagereq) / 100.0),
					Usedcores: uint(val.Coresreq),
				})
				score += DCUScore(schedulePolicy, ds.Device[i])
			}
			ctrdevs = append(ctrdevs, devs)
			continue
		}

	loop:
		for i := len(ds.Device) - 1; i >= 0; i-- {
			klog.V(3).InfoS("Scoring pod request", "memReq", val.Memreq, "memPercentageReq", val.MemPercentagereq, "coresReq", val.Coresreq, "Nums", val.Nums, "Index", i, "ID", ds.Device[i].ID)
			klog.V(3).InfoS("Current Device", "Index", i, "TotalMemory", ds.Device[i].Memory, "UsedMemory", ds.Device[i].UsedMem, "UsedCores", ds.Device[i].UsedCore, "replicate", replicate)
			if ds.Device[i].Number <= uint(ds.Device[i].UsedNum) {
				continue
			}
			memreqForCard := uint(0)
			// if we have mempercentage request, we ignore the mem request for every cards
			if val.MemPercentagereq != 101 {
				memreqForCard = uint(float64(ds.Device[i].Memory) * float64(val.MemPercentagereq) / 100.0)
			} else {
				memreqForCard = uint(val.Memreq * int32(getConfig().DeviceMemoryScaling))
			}
			if int(ds.Device[i].Memory)-int(ds.Device[i].UsedMem) < int(memreqForCard) {
				continue
			}
			if ds.Device[i].UsedCore+uint(val.Coresreq) > 100 {
				continue
			}
			// Coresreq=100 indicates it want this card exclusively
			if val.Coresreq == 100 && ds.Device[i].UsedNum > 0 {
				continue
			}
			// You can't allocate core=0 job to an already full GPU
			if ds.Device[i].UsedCore == 100 && val.Coresreq == 0 {
				continue
			}
			if !checkType(pod.Annotations, *ds.Device[i], val) {
				klog.Errorln("failed checktype", ds.Device[i].Type, val.Type)
				continue
			}
			if !checkDCUUUID(pod.Annotations, ds.Device[i].UUID) {
				klog.Errorln("failed check UUID", ds.Device[i].UUID)
				continue
			}

			fit, uuid := ds.TryAddPod(ds.Device[i], memreqForCard, uint(val.Coresreq))

			if !fit {
				klog.V(3).Info(ds.Device[i].ID, "not fit")
				continue
			}
			//total += gs.Devices[i].Count
			//free += node.Devices[i].Count - node.Devices[i].Used
			if val.Nums > 0 {
				val.Nums--
				klog.V(3).Info("fitted uuid: ", uuid)
				devs = append(devs, ContainerDevice{
					UUID:      uuid,
					Type:      val.Type,
					Usedmem:   memreqForCard,
					Usedcores: uint(val.Coresreq),
				})
				score += DCUScore(schedulePolicy, ds.Device[i])
			}
			if val.Nums == 0 {
				break
			}
		}
		if val.Nums > 0 {
			return false, []ContainerDevices{}, 0, fmt.Errorf("not enough dcu fitted on this node")
		}
		ctrdevs = append(ctrdevs, devs)
	}
	return true, ctrdevs, score, nil
}

func DCUScore(schedulePolicy string, device *DCUDevice) float64 {
	var score float64
	switch schedulePolicy {
	case binpackPolicy:
		score = binpackMultiplier * (float64(device.UsedMem) / float64(device.Memory))
	case spreadPolicy:
		if device.UsedNum == 1 {
			score = spreadMultiplier
		}
	default:
		score = float64(0)
	}
	return score
}

func patchPodAnnotations(kubeClient kubernetes.Interface, pod *v1.Pod, annotations map[string]string) error {
	type patchMetadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	}
	type patchPod struct {
		Metadata patchMetadata `json:"metadata"`
		//Spec     patchSpec     `json:"spec,omitempty"`
	}

	p := patchPod{}
	p.Metadata.Annotations = annotations

	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = kubeClient.CoreV1().Pods(pod.Namespace).
		Patch(context.Background(), pod.Name, k8stypes.StrategicMergePatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("patch pod %v failed, %v", pod.Name, err)
	}

	return err
}

func patchNodeAnnotations(node *v1.Node, annotations map[string]string) error {
	type patchMetadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	}
	type patchNode struct {
		Metadata patchMetadata `json:"metadata"`
		//Spec     patchSpec     `json:"spec,omitempty"`
	}

	p := patchNode{}
	p.Metadata.Annotations = annotations

	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = devices.GetClient().CoreV1().Nodes().
		Patch(context.Background(), node.Name, k8stypes.StrategicMergePatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("patch node %v failed, %v", node.Name, err)
	}
	return err
}

func getConfig() config.HygonConfig {
	if config.GetConfig() != nil {
		return config.GetConfig().HygonConfig
	}
	return config.GetDefaultDevicesConfig().HygonConfig
}

func ParseTopologyInfoConfigMap(cm *v1.ConfigMap) (map[string]*Topology, error) {
	result := make(map[string]*Topology)

	for nodeName, topologyJSON := range cm.Data {
		var topo Topology
		if err := json.Unmarshal([]byte(topologyJSON), &topo); err != nil {
			return nil, fmt.Errorf("node %s parse failed: %w", nodeName, err)
		}
		result[nodeName] = &topo
	}
	return result, nil
}

func GetTopologyInfo(nodeName string) (*Topology, error) {
	cm, err := devices.GetClient().CoreV1().ConfigMaps("kube-system").
		Get(context.Background(), "dcu-topology-info", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	topos, err := ParseTopologyInfoConfigMap(cm)
	if err != nil {
		return nil, err
	}

	if topo, ok := topos[nodeName]; ok {
		return topo, nil
	}
	return nil, nil
}

func FilterTopologyByAllocated(topo *Topology, allocatedIDs []int) *Topology {
	if topo == nil || len(topo.Matrix) == 0 {
		return topo
	}

	// Build the allocated set
	alloc := make(map[int]struct{}, len(allocatedIDs))
	for _, id := range allocatedIDs {
		alloc[id] = struct{}{}
	}

	// Find the row indices of the matrix that need to be retained.
	keepIdx := make([]int, 0, len(topo.Matrix))
	for i := 0; i < len(topo.Matrix); i++ {
		devID := topo.Matrix[i][i].SrcDvInd
		if _, ok := alloc[devID]; !ok {
			keepIdx = append(keepIdx, i)
		}
	}

	// Construct a new matrix
	newMatrix := make([][]Link, len(keepIdx))
	for i, oldI := range keepIdx {
		newMatrix[i] = make([]Link, len(keepIdx))
		for j, oldJ := range keepIdx {
			newMatrix[i][j] = topo.Matrix[oldI][oldJ]
		}
	}

	return &Topology{
		Matrix: newMatrix,
	}
}

func SelectBestDevices(topo *Topology, k int) []int {
	n := len(topo.Matrix)
	if k <= 0 || k > n {
		return nil
	}

	bestScore := -1 << 30
	var bestIdx []int

	var dfs func(start int, chosen []int, score int)
	dfs = func(start int, chosen []int, score int) {
		if len(chosen) == k {
			if score > bestScore {
				bestScore = score
				bestIdx = append([]int{}, chosen...)
			}
			return
		}

		if n-start < k-len(chosen) {
			return
		}

		for i := start; i < n; i++ {
			add := 0
			for _, j := range chosen {
				add += topo.Matrix[j][i].Weight
			}
			dfs(i+1, append(chosen, i), score+add)
		}
	}

	dfs(0, nil, 0)

	// Convert to the real device index (SrcDvInd)
	result := make([]int, 0, k)
	for _, idx := range bestIdx {
		result = append(result, topo.Matrix[idx][idx].SrcDvInd)
	}
	return result
}
