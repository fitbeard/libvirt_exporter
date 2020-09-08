// Copyright 2017 Kumina, https://kumina.nl/
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Project forked from https://github.com/kumina/libvirt_exporter
// And then forked from https://github.com/rumanzo/libvirt_exporter_improved

package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/fitbeard/libvirt_exporter/libvirtSchema"
	"github.com/libvirt/libvirt-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	libvirtUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "", "up"),
		"Whether scraping libvirt's metrics was successful.",
		nil,
		nil)

	libvirtDomainInfoMetaDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "meta"),
		"Domain metadata",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainInfoMaxMemBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "maximum_memory_bytes"),
		"Maximum allowed memory of the domain, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainInfoMemoryUsageBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "memory_usage_bytes"),
		"Memory usage of the domain, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainInfoNrVirtCPUDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "virtual_cpus"),
		"Number of virtual CPUs for the domain.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainInfoCPUTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "cpu_time_seconds_total"),
		"Amount of CPU time used by the domain, in seconds.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainInfoVirDomainState = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_info", "vstate"),
		"Virtual domain state. 0: no state, 1: the domain is running, 2: the domain is blocked on resource,"+
			" 3: the domain is paused by user, 4: the domain is being shut down, 5: the domain is shut off,"+
			"6: the domain is crashed, 7: the domain is suspended by guest power management",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)

	libvirtDomainMetaBlockDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "meta"),
		"Block device metadata info. Device name, source file, serial.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockRdBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "read_bytes_total"),
		"Number of bytes read from a block device, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockRdReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "read_requests_total"),
		"Number of read requests from a block device.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockRdTotalTimeSecondsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "read_time_seconds_total"),
		"Total time spent on reads from a block device, in seconds.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockWrBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "write_bytes_total"),
		"Number of bytes written to a block device, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockWrReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "write_requests_total"),
		"Number of write requests to a block device.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockWrTotalTimesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "write_time_seconds_total"),
		"Total time spent on writes on a block device, in seconds",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockFlushReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "flush_requests_total"),
		"Total flush requests from a block device.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockFlushTotalTimeSecondsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "flush_time_seconds_total"),
		"Total time in seconds spent on cache flushing to a block device",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockAllocationDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "allocation"),
		"Offset of the highest written sector on a block device.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockCapacityBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "capacity_bytes"),
		"Logical size in bytes of the block device	backing image.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainBlockPhysicalSizeBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block_stats", "physicalsize_bytes"),
		"Physical size in bytes of the container of the backing image.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"},
		nil)
	libvirtDomainMetaInterfacesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface", "meta"),
		"Interfaces metadata. Source bridge, target device, interface uuid",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceRxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "receive_bytes_total"),
		"Number of bytes received on a network interface, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceRxPacketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "receive_packets_total"),
		"Number of packets received on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceRxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "receive_errors_total"),
		"Number of packet receive errors on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceRxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "receive_drops_total"),
		"Number of packet receive drops on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceTxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "transmit_bytes_total"),
		"Number of bytes transmitted on a network interface, in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceTxPacketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "transmit_packets_total"),
		"Number of packets transmitted on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceTxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "transmit_errors_total"),
		"Number of packet transmit errors on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)
	libvirtDomainInterfaceTxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_interface_stats", "transmit_drops_total"),
		"Number of packet transmit drops on a network interface.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid", "target_device", "source_bridge", "virtual_interface"},
		nil)

	libvirtDomainMemoryStatMajorFaultTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "major_fault_total"),
		"Page faults occur when a process makes a valid access to virtual memory that is not available. "+
			"When servicing the page fault, if disk IO is required, it is considered a major fault.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatMinorFaultTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "minor_fault_total"),
		"Page faults occur when a process makes a valid access to virtual memory that is not available. "+
			"When servicing the page not fault, if disk IO is required, it is considered a minor fault.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatUnusedBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "unused_bytes"),
		"The amount of memory left completely unused by the system. Memory that is available but used for "+
			"reclaimable caches should NOT be reported as free. This value is expressed in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatAvailableBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "available_bytes"),
		"The total amount of usable memory as seen by the domain. This value may be less than the amount of "+
			"memory assigned to the domain if a balloon driver is in use or if the guest OS does not initialize all "+
			"assigned pages. This value is expressed in bytes.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatActualBaloonBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "actual_balloon_bytes"),
		"Current balloon value (in bytes).",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatRssBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "rss_bytes"),
		"Resident Set Size of the process running the domain. This value is in bytes",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatUsableBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "usable_bytes"),
		"How much the balloon can be inflated without pushing the guest system to swap, corresponds "+
			"to 'Available' in /proc/meminfo",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatDiskCachesBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "disk_cache_bytes"),
		"The amount of memory, that can be quickly reclaimed without additional I/O (in bytes)."+
			"Typically these pages are used for caching files from disk.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)
	libvirtDomainMemoryStatUsedPercentDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory_stats", "used_percent"),
		"The amount of memory in percent, that used by domain.",
		[]string{"domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"},
		nil)

	projectFilter *string
)

// CollectDomain extracts Prometheus metrics from a libvirt domain.
func CollectDomain(ch chan<- prometheus.Metric, stat libvirt.DomainStats) error {
	domainName, err := stat.Domain.GetName()
	if err != nil {
		return err
	}

	domainUUID, err := stat.Domain.GetUUIDString()
	if err != nil {
		return err
	}

	// Decode XML description of domain to get block device names, etc.
	xmlDesc, err := stat.Domain.GetXMLDesc(0)
	if err != nil {
		return err
	}
	var desc libvirtSchema.Domain
	err = xml.Unmarshal([]byte(xmlDesc), &desc)
	if err != nil {
		return err
	}

	match, _ := regexp.MatchString(*projectFilter, desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName)
	if match {
		return nil
	}
	// Report domain info.
	info, err := stat.Domain.GetInfo()
	if err != nil {
		return err
	}
	// "domain", "uuid", "nova_name", "flavor", "user_name", "user_uuid", "project_name", "project_uuid", "root_type", "root_uuid"
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoMetaDesc,
		prometheus.GaugeValue,
		float64(1),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoMaxMemBytesDesc,
		prometheus.GaugeValue,
		float64(info.MaxMem)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoMemoryUsageBytesDesc,
		prometheus.GaugeValue,
		float64(info.Memory)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoNrVirtCPUDesc,
		prometheus.GaugeValue,
		float64(info.NrVirtCpu),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoCPUTimeDesc,
		prometheus.CounterValue,
		float64(info.CpuTime)/1000/1000/1000, // From nsec to sec
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainInfoVirDomainState,
		prometheus.GaugeValue,
		float64(info.State),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	// Report block device statistics.
	for _, disk := range stat.Block {
		var DiskSource string
		var Device *libvirtSchema.Disk
		if disk.Name == "hdc" {
			continue
		}
		/*  "block.<num>.path" - string describing the source of block device <num>,
		    if it is a file or block device (omitted for network
		    sources and drives with no media inserted). For network device (i.e. rbd) take from xml. */
		for _, dev := range desc.Devices.Disks {
			if dev.Target.Device == disk.Name {
				if disk.PathSet {
					DiskSource = disk.Path

				} else {
					DiskSource = dev.Source.Name
				}
				Device = &dev
				break
			}
		}
		// "target_device", "source_file", "serial", "bus", "disk_type", "driver_type", "cache", "discard"
		ch <- prometheus.MustNewConstMetric(
			libvirtDomainMetaBlockDesc,
			prometheus.GaugeValue,
			float64(1),
			domainName,
			domainUUID,
			desc.Metadata.NovaInstance.NovaName,
			desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
			desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
			desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
			desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
			desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
			desc.Metadata.NovaInstance.NovaRoot.RootType,
			desc.Metadata.NovaInstance.NovaRoot.RootUUID,
			disk.Name,
			DiskSource,
			Device.Serial,
			Device.Target.Bus,
			Device.DiskType,
			Device.Driver.Type,
			Device.Driver.Cache,
			Device.Driver.Discard,
		)

		// https://libvirt.org/html/libvirt-libvirt-domain.html#virConnectGetAllDomainStats
		if disk.RdBytesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockRdBytesDesc,
				prometheus.CounterValue,
				float64(disk.RdBytes),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.RdReqsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockRdReqDesc,
				prometheus.CounterValue,
				float64(disk.RdReqs),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.RdTimesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockRdTotalTimeSecondsDesc,
				prometheus.CounterValue,
				float64(disk.RdTimes)/1e9,
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.WrBytesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockWrBytesDesc,
				prometheus.CounterValue,
				float64(disk.WrBytes),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.WrReqsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockWrReqDesc,
				prometheus.CounterValue,
				float64(disk.WrReqs),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.WrTimesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockWrTotalTimesDesc,
				prometheus.CounterValue,
				float64(disk.WrTimes)/1e9,
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.FlReqsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockFlushReqDesc,
				prometheus.CounterValue,
				float64(disk.FlReqs),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.FlTimesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockFlushTotalTimeSecondsDesc,
				prometheus.CounterValue,
				float64(disk.FlTimes)/1e9,
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.AllocationSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockAllocationDesc,
				prometheus.GaugeValue,
				float64(disk.Allocation),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.CapacitySet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockCapacityBytesDesc,
				prometheus.GaugeValue,
				float64(disk.Capacity),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
		if disk.PhysicalSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainBlockPhysicalSizeBytesDesc,
				prometheus.GaugeValue,
				float64(disk.Physical),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				disk.Name,
				DiskSource,
				Device.Serial,
				Device.Target.Bus,
				Device.DiskType,
				Device.Driver.Type,
				Device.Driver.Cache,
				Device.Driver.Discard)
		}
	}

	// Report network interface statistics.
	for _, iface := range stat.Net {
		var SourceBridge string
		var VirtualInterface string
		// Additional info for ovs network
		for _, net := range desc.Devices.Interfaces {
			if net.Target.Device == iface.Name {
				SourceBridge = net.Source.Bridge
				VirtualInterface = net.Virtualport.Parameters.InterfaceID
				break
			}
		}
		// "target_device", "source_bridge", "virtual_interface"
		ch <- prometheus.MustNewConstMetric(
			libvirtDomainMetaInterfacesDesc,
			prometheus.GaugeValue,
			float64(1),
			domainName,
			domainUUID,
			desc.Metadata.NovaInstance.NovaName,
			desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
			desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
			desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
			desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
			desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
			desc.Metadata.NovaInstance.NovaRoot.RootType,
			desc.Metadata.NovaInstance.NovaRoot.RootUUID,
			iface.Name,
			SourceBridge,
			VirtualInterface)
		if iface.RxBytesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceRxBytesDesc,
				prometheus.CounterValue,
				float64(iface.RxBytes),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.RxPktsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceRxPacketsDesc,
				prometheus.CounterValue,
				float64(iface.RxPkts),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.RxErrsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceRxErrsDesc,
				prometheus.CounterValue,
				float64(iface.RxErrs),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.RxDropSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceRxDropDesc,
				prometheus.CounterValue,
				float64(iface.RxDrop),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.TxBytesSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceTxBytesDesc,
				prometheus.CounterValue,
				float64(iface.TxBytes),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.TxPktsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceTxPacketsDesc,
				prometheus.CounterValue,
				float64(iface.TxPkts),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.TxErrsSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceTxErrsDesc,
				prometheus.CounterValue,
				float64(iface.TxErrs),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
		if iface.TxDropSet {
			ch <- prometheus.MustNewConstMetric(
				libvirtDomainInterfaceTxDropDesc,
				prometheus.CounterValue,
				float64(iface.TxDrop),
				domainName,
				domainUUID,
				desc.Metadata.NovaInstance.NovaName,
				desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
				desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
				desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
				desc.Metadata.NovaInstance.NovaRoot.RootType,
				desc.Metadata.NovaInstance.NovaRoot.RootUUID,
				iface.Name,
				SourceBridge,
				VirtualInterface)
		}
	}

	// Collect Memory Stats
	memorystat, err := stat.Domain.MemoryStats(11, 0)
	var MemoryStats libvirtSchema.VirDomainMemoryStats
	var usedPercent float64
	if err == nil {
		MemoryStats = memoryStatCollect(&memorystat)
		if MemoryStats.Usable != 0 && MemoryStats.Available != 0 {
			usedPercent = (float64(MemoryStats.Available) - float64(MemoryStats.Usable)) / (float64(MemoryStats.Available) / float64(100))
		}

	}
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatMajorFaultTotalDesc,
		prometheus.CounterValue,
		float64(MemoryStats.MajorFault),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatMinorFaultTotalDesc,
		prometheus.CounterValue,
		float64(MemoryStats.MinorFault),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatUnusedBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.Unused)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatAvailableBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.Available)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatActualBaloonBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.ActualBalloon)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatRssBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.Rss)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatUsableBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.Usable)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatDiskCachesBytesDesc,
		prometheus.GaugeValue,
		float64(MemoryStats.DiskCaches)*1024,
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)
	ch <- prometheus.MustNewConstMetric(
		libvirtDomainMemoryStatUsedPercentDesc,
		prometheus.GaugeValue,
		float64(usedPercent),
		domainName,
		domainUUID,
		desc.Metadata.NovaInstance.NovaName,
		desc.Metadata.NovaInstance.NovaFlavor.FlavorName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserName,
		desc.Metadata.NovaInstance.NovaOwner.NovaUser.UserUUID,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectName,
		desc.Metadata.NovaInstance.NovaOwner.NovaProject.ProjectUUID,
		desc.Metadata.NovaInstance.NovaRoot.RootType,
		desc.Metadata.NovaInstance.NovaRoot.RootUUID)

	return nil
}

// CollectFromLibvirt obtains Prometheus metrics from all domains in a
// libvirt setup.
func CollectFromLibvirt(ch chan<- prometheus.Metric, uri string) error {
	conn, err := libvirt.NewConnectReadOnly(uri)
	if err != nil {
		return err
	}
	defer conn.Close()

	stats, err := conn.GetAllDomainStats([]*libvirt.Domain{}, libvirt.DOMAIN_STATS_STATE|libvirt.DOMAIN_STATS_CPU_TOTAL|
		libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BALLOON|libvirt.DOMAIN_STATS_BLOCK|
		libvirt.DOMAIN_STATS_PERF|libvirt.DOMAIN_STATS_VCPU,
		//libvirt.CONNECT_GET_ALL_DOMAINS_STATS_NOWAIT, // maybe in future
		libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING|libvirt.CONNECT_GET_ALL_DOMAINS_STATS_SHUTOFF)
	defer func(stats []libvirt.DomainStats) {
		for _, stat := range stats {
			stat.Domain.Free()
		}
	}(stats)
	if err != nil {
		return err
	}
	for _, stat := range stats {
		err = CollectDomain(ch, stat)
		if err != nil {
			log.Printf("Failed to scrape metrics: %s", err)
		}
	}
	return nil
}

func memoryStatCollect(memorystat *[]libvirt.DomainMemoryStat) libvirtSchema.VirDomainMemoryStats {
	var MemoryStats libvirtSchema.VirDomainMemoryStats
	for _, domainmemorystat := range *memorystat {
		switch tag := domainmemorystat.Tag; tag {
		case 2:
			MemoryStats.MajorFault = domainmemorystat.Val
		case 3:
			MemoryStats.MinorFault = domainmemorystat.Val
		case 4:
			MemoryStats.Unused = domainmemorystat.Val
		case 5:
			MemoryStats.Available = domainmemorystat.Val
		case 6:
			MemoryStats.ActualBalloon = domainmemorystat.Val
		case 7:
			MemoryStats.Rss = domainmemorystat.Val
		case 8:
			MemoryStats.Usable = domainmemorystat.Val
		case 10:
			MemoryStats.DiskCaches = domainmemorystat.Val
		}
	}
	return MemoryStats
}

// LibvirtExporter implements a Prometheus exporter for libvirt state.
type LibvirtExporter struct {
	uri string
}

// NewLibvirtExporter creates a new Prometheus exporter for libvirt.
func NewLibvirtExporter(uri string) (*LibvirtExporter, error) {
	return &LibvirtExporter{
		uri: uri,
	}, nil
}

// Describe returns metadata for all Prometheus metrics that may be exported.
func (e *LibvirtExporter) Describe(ch chan<- *prometheus.Desc) {
	// Status
	ch <- libvirtUpDesc

	// Domain info
	ch <- libvirtDomainInfoMetaDesc
	ch <- libvirtDomainInfoMaxMemBytesDesc
	ch <- libvirtDomainInfoMemoryUsageBytesDesc
	ch <- libvirtDomainInfoNrVirtCPUDesc
	ch <- libvirtDomainInfoCPUTimeDesc
	ch <- libvirtDomainInfoVirDomainState

	// Domain block stats
	ch <- libvirtDomainMetaBlockDesc
	ch <- libvirtDomainBlockRdBytesDesc
	ch <- libvirtDomainBlockRdReqDesc
	ch <- libvirtDomainBlockRdTotalTimeSecondsDesc
	ch <- libvirtDomainBlockWrBytesDesc
	ch <- libvirtDomainBlockWrReqDesc
	ch <- libvirtDomainBlockWrTotalTimesDesc
	ch <- libvirtDomainBlockFlushReqDesc
	ch <- libvirtDomainBlockFlushTotalTimeSecondsDesc
	ch <- libvirtDomainBlockAllocationDesc
	ch <- libvirtDomainBlockCapacityBytesDesc
	ch <- libvirtDomainBlockPhysicalSizeBytesDesc

	// Domain net interfaces stats
	ch <- libvirtDomainMetaInterfacesDesc
	ch <- libvirtDomainInterfaceRxBytesDesc
	ch <- libvirtDomainInterfaceRxPacketsDesc
	ch <- libvirtDomainInterfaceRxErrsDesc
	ch <- libvirtDomainInterfaceRxDropDesc
	ch <- libvirtDomainInterfaceTxBytesDesc
	ch <- libvirtDomainInterfaceTxPacketsDesc
	ch <- libvirtDomainInterfaceTxErrsDesc
	ch <- libvirtDomainInterfaceTxDropDesc

	// Domain memory stats
	ch <- libvirtDomainMemoryStatMajorFaultTotalDesc
	ch <- libvirtDomainMemoryStatMinorFaultTotalDesc
	ch <- libvirtDomainMemoryStatUnusedBytesDesc
	ch <- libvirtDomainMemoryStatAvailableBytesDesc
	ch <- libvirtDomainMemoryStatActualBaloonBytesDesc
	ch <- libvirtDomainMemoryStatRssBytesDesc
	ch <- libvirtDomainMemoryStatUsableBytesDesc
	ch <- libvirtDomainMemoryStatDiskCachesBytesDesc
}

// Collect scrapes Prometheus metrics from libvirt.
func (e *LibvirtExporter) Collect(ch chan<- prometheus.Metric) {
	err := CollectFromLibvirt(ch, e.uri)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			libvirtUpDesc,
			prometheus.GaugeValue,
			1.0)
	} else {
		log.Printf("Failed to scrape metrics: %s", err)
		ch <- prometheus.MustNewConstMetric(
			libvirtUpDesc,
			prometheus.GaugeValue,
			0.0)
	}
}

func main() {
	var (
		app           = kingpin.New("libvirt_exporter", "Prometheus metrics exporter for libvirt")
		listenAddress = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9177").String()
		metricsPath   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		libvirtURI    = app.Flag("libvirt.uri", "Libvirt URI from which to extract metrics.").Default("qemu:///system").String()
	)

	projectFilter = app.Flag("filter.project", "Regular expression matching project names that should be filtered out.").Default("^$").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	exporter, err := NewLibvirtExporter(*libvirtURI)
	if err != nil {
		panic(err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Libvirt Exporter</title></head>
			<body>
			<h1>Libvirt Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
