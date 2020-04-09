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

package libvirt_schema

type Domain struct {
	Devices  Devices  `xml:"devices"`
	Metadata Metadata `xml:"metadata"`
	UUID     string   `xml:"uuid"`
}

type Metadata struct {
	// The actual xml tag is nova:instance, but we don't care about the namespaces
	NovaInstance NovaInstance `xml:"instance"`
}

type NovaInstance struct {
	Name   string     `xml:"name"`
	Flavor NovaFlavor `xml:"flavor"`
	Owner  NovaOwner  `xml:"owner"`
}

type NovaFlavor struct {
	Name string `xml:"name,attr"`
}

type NovaOwner struct {
	Text    string   `xml:",chardata"`
	User    struct {
		Text string `xml:",chardata"`
		Uuid string `xml:"uuid,attr"`
	} `xml:"user"`
	Project struct {
		Text string `xml:",chardata"`
		Uuid string `xml:"uuid,attr"`
	} `xml:"project"`
}

type Devices struct {
	Disks      []Disk      `xml:"disk"`
	Interfaces []Interface `xml:"interface"`
}

type Disk struct {
	Type   string     `xml:"type,attr"`
	Device string     `xml:"device,attr"`
	Source DiskSource `xml:"source"`
	Target DiskTarget `xml:"target"`
}

type DiskSource struct {
	File string `xml:"file,attr"`
	Dev string `xml:"dev,attr"`
	Protocol string `xml:"protocol,attr"`
	Name string `xml:"name,attr"`
}

type DiskTarget struct {
	Device string `xml:"dev,attr"`
}

type Interface struct {
	Source InterfaceSource `xml:"source"`
	Target InterfaceTarget `xml:"target"`
}

type InterfaceSource struct {
	Bridge string `xml:"bridge,attr"`
}

type InterfaceTarget struct {
	Device string `xml:"dev,attr"`
}
