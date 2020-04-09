# Prometheus libvirt exporter

It's a fork of [libvirt_exporter](https://github.com/kumina/libvirt_exporter).

This version returns more data, which is needed in our case.

### The following metrics are being added

- libvirt_domains_number

> Domain (instance) count.

- libvirt_domain_state_code

> Domain status code running/stopped.

### The following labels are being added

- disk_type

> Disk type like file (qcow2), network (rbd, glusterfs) and block (iscsi, san).

- source_dev

> Disk path like /dev/disk/by-id/78da994c65700812-5a2895de000000d0. Only for disk_type block.

- source_name

> Disk and pool name like ceph-pool-name/volume-358fba2f-19cf-4339-b1cc-e8e9d975a3cb. Only for disk_type network.

- source_protocol

> Disk network protocol like rbd. Only for disk_type network.

With the `--libvirt.export-nova-metadata` flag, it will export the following additional OpenStack-specific labels for every domain:

- project_uuid

> OpenStack project uuid.

- project_name

> Same as in original. OpenStack project name.

- nova_name

> Renamed from name for back compatibility. OpenStack instance name.

- flavor

> Same as in original. OpenStack flavor name.
