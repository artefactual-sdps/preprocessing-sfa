# preprocessing-sfa

**preprocessing-sfa** is an Enduro preprocessing workflow for SFA SIPs.

- [Configuration](#configuration)
- [Local environment](#local-environment)
- [Makefile](#makefile)

## Configuration

The preprocessing workers need to share the filesystem with Enduro's a3m or
Archivematica workers. They must be connected to the same Temporal server
and related to each other with the namespace, task queue and workflow name.

### Preprocessing:

The entire configuration for the preprocessing worker:

```toml
debug = false
verbosity = 0
sharedPath = "/home/enduro/preprocessing"

[temporal]
address = "temporal.enduro-sdps:7233"
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"

[worker]
maxConcurrentSessions = 1
```

### Enduro:

The preprocessing section for Enduro's configuration:

```toml
[preprocessing]
enabled = true
extract = true
sharedPath = "/home/enduro/preprocessing"

[preprocessing.temporal]
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"
```

## Local environment

### Requirements

This project uses Tilt to set up a local environment building the Docker images
in a Kubernetes cluster. It has been tested with k3d, Minikube and Kind.

- [Docker] (v18.09+)
- [kubectl]
- [Tilt] (v0.22.2+)

A local Kubernetes cluster:

- [k3d] _(recomended, used in CI)_
- [Minikube] _(tested)_
- [Kind] _(tested)_

It can run with other solutions like Microk8s or Docker for Desktop/Mac and
even against remote clusters, check Tilt's [Choosing a Local Dev Cluster] and
[Install] documentation for more information to install these requirements.

Additionally, follow the [Manage Docker as a non-root user] post-install guide
so that you don’t have to run Tilt with `sudo`. _Note that managing Docker as a
non-root user is **different** from running the docker daemon as a non-root user
(rootless)._

### Requirements for development

While we run the services inside a Kubernetes cluster we recomend to install
Go and other tools locally to ease the development process.

- [Go] (1.22+)
- GNU [Make] and [GCC]

### Set up

Start a local Kubernetes cluster with a local registry. For example, with k3d:

```bash
k3d cluster create preprocessing --registry-create sdps-registry
```

Or using an existing registry:

```bash
k3d cluster create preprocessing --registry-use sdps-registry
```

Make sure kubectl is available and configured to use that cluster:

```bash
kubectl config view
```

Clone this repository and move into its folder if you have not done that
previously:

```bash
git clone git@github.com:artefactual-sdps/preprocessing-sfa.git
cd preprocessing-sfa
```

Bring up the environment:

```bash
tilt up
```

While the Docker images are built/downloaded and the Kubernetes resources are
created, hit `space` to open the Tilt UI in your browser. Check the [Tilt UI]
documentation to know more about it.

### Live updates

Tilt, by default, will watch for file changes in the project folder and it will
sync those changes, rebuild the Docker images and recreate the resources when
necessary. However, we have _disabled_ auto-load within the Tiltfile to reduce
the use of hardware resources. There are refresh buttons on each resource in the
Tilt UI that allow triggering manual updates and re-executing jobs and local
resources. You can also set the `trigger_mode` env string to `TRIGGER_MODE_AUTO`
within your local `.tilt.env` file to override this change and enable auto mode.

### Stop/start the environment

Run `ctrl-c` on the terminal where `tilt up` is running and stop the cluster
with:

```bash
k3d cluster stop preprocessing
```

To start the environment again:

```bash
k3d cluster start preprocessing
tilt up
```

### Clear the cluster

> Check the Tilt UI helpers below to just flush the existing data.

To remove the resources created by Tilt in the cluster, execute:

```bash
tilt down
```

Note that it will take some time to delete the persistent volumes when you
run `tilt down` and flushing the existing data does not delete the cluster.
To delete the volumes immediately, you can delete the cluster.

### Delete the cluster

Deleting the cluster will remove all the resources immediatly, deleting
cluster container from the host. With k3d, run:

```bash
k3d cluster delete preprocessing
```

### Tilt environment configuration

A few configuration options can be changed by having a `.tilt.env` file
located in the root of the project. Example:

```text
TRIGGER_MODE_AUTO=true
```

#### TRIGGER_MODE_AUTO

Enables live updates on code changes for the preprocessing worker.

### Tilt UI helpers

#### Submit

In the Tilt UI header there is a cloud icon/button that allows to trigger the
preprocessing workflow. Click the caret to set the path to a file/directory in
the host, then click the cloud icon to trigger the workflow.

#### Flush

Also in the Tilt UI header, click the trash button to flush the existing data.
This will recreate the MySQL databases and restart the required resources.

## Makefile

The Makefile provides developer utility scripts via command line `make` tasks.
Running `make` with no arguments (or `make help`) prints the help message.
Dependencies are downloaded automatically.

### Debug mode

The debug mode produces more output, including the commands executed. E.g.:

```shell
$ make env DBG_MAKEFILE=1
Makefile:10: ***** starting Makefile for goal(s) "env"
Makefile:11: ***** Fri 10 Nov 2023 11:16:16 AM CET
go env
GO111MODULE=''
GOARCH='amd64'
...
```

[enduro documentation]: https://github.com/artefactual-sdps/enduro/blob/main/docs/src/dev-manual/preprocessing.md
[docker]: https://docs.docker.com/get-docker/
[kubectl]: https://kubernetes.io/docs/tasks/tools/#kubectl
[tilt]: https://docs.tilt.dev/tutorial/1-prerequisites.html#install-tilt
[k3d]: https://k3d.io/v5.4.3/#installation
[minikube]: https://minikube.sigs.k8s.io/docs/start/
[kind]: https://kind.sigs.k8s.io/docs/user/quick-start#installation
[choosing a local dev cluster]: https://docs.tilt.dev/choosing_clusters.html
[install]: https://docs.tilt.dev/install.html
[manage docker as a non-root user]: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user
[tilt ui]: https://docs.tilt.dev/tutorial/3-tilt-ui.html
[go]: https://go.dev/doc/install
[make]: https://www.gnu.org/software/make/
[gcc]: https://gcc.gnu.org/
