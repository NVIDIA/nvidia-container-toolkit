# NVIDIA CDI Hook

The CLI `nvidia-cdi-hook` provides container device runtime hook capabilities when
called by a container runtime, as specific in a
[Container Device Interface](https://tags.cncf.io/container-device-interface/blob/main/SPEC.md)
file.

## Generating a CDI

The CDI itself is created for an NVIDIA-capable device using the
[`nvidia-ctk cdi generate`](../nvidia-ctk/) command.

When `nvidia-ctk cdi generate` is run, the CDI specification is generated as a yaml file.
The CDI specification provides instructions for a container runtime to set up devices, files and
other resources for the container prior to starting it. Those instructions
may include executing command-line tools to prepare the filesystem. The execution
of such command-line tools is called a hook.

`nvidia-cdi-hook` is the CLI tool that is expected to be called by the container runtime,
when specified by the CDI file.

See the [`nvidia-ctk` documentation](../nvidia-ctk/README.md) for more information
on generating a CDI file.

## Functionality

The `nvidia-cdi-hook` CLI provides the following functionality:

* `chmod` - Change the permissions of a file or directory inside the directory path to be mounted into a container.
* `create-symlinks` - Create symlinks inside the directory path to be mounted into a container.
* `update-ldcache` - Update the dynamic linker cache inside the directory path to be mounted into a container.
