package ash

import (
	"bytes"
	"fmt"
	"math"
	"runtime/metrics"
	"time"

	vk "github.com/tomas-mraz/vulkan"
)

const (
	end     = "\x00"
	endChar = '\x00'
)

var (
	rtExtensions = [...]string{
		MakeCString(vk.KhrAccelerationStructureExtensionName),
		MakeCString(vk.KhrRayTracingPipelineExtensionName),
		MakeCString(vk.KhrBufferDeviceAddressExtensionName),
		MakeCString(vk.KhrDeferredHostOperationsExtensionName),
		MakeCString(vk.ExtDescriptorIndexingExtensionName),
		MakeCString(vk.KhrSpirv14ExtensionName),
		MakeCString(vk.KhrShaderFloatControlsExtensionName),
	}
)

func trimCString(slice []byte) string {
	return string(bytes.TrimRight(slice, end))
}

func MakeCString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func RaytracingExtensions() []string {
	return append([]string(nil), rtExtensions[:]...)
}

// LoadShaderFromBytes creates a shader module from raw SPIR-V bytecode.
func LoadShaderFromBytes(device vk.Device, data []byte) (vk.ShaderModule, error) {
	var module vk.ShaderModule
	shaderModuleCreateInfo := vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint64(len(data)),
		PCode:    repackUint32(data),
	}
	err := vk.Error(vk.CreateShaderModule(device, &shaderModuleCreateInfo, nil, &module))
	if err != nil {
		err = fmt.Errorf("vk.CreateShaderModule failed with %s", err)
		return module, err
	}
	return module, nil
}

func printGCPauses() {
	samples := make([]metrics.Sample, 2)
	samples[0].Name = "/gc/cycles/total:gc-cycles"
	samples[1].Name = "/sched/pauses/total/gc:seconds"

	metrics.Read(samples)

	cycles := samples[0].Value.Uint64()
	hist := samples[1].Value.Float64Histogram()

	var totalPauses uint64
	for i := 0; i < len(hist.Counts); i++ {
		low := hist.Buckets[i]
		if !math.IsInf(low, -1) {
			low = low * 1000000
		}
		high := hist.Buckets[i+1]
		if !math.IsInf(high, 1) {
			high = high * 1000000
		}
		count := hist.Counts[i]
		if count > 0 {
			fmt.Printf("GC histogram range [%4.0f - %4.0f µs]: %d×\n", low, high, count)
		}
		totalPauses += count
	}
	fmt.Printf("GC total cycles: %d pauses: %d\n", cycles, totalPauses)
}

// StartPrintGCPauses every 10 seconds prints to std output information about GC
func StartPrintGCPauses(period time.Duration) {
	if period < 10*time.Second {
		period = 10 * time.Second
	}
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				printGCPauses()
			}
		}
	}()
}
