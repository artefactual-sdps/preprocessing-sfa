# preprocessing-sfa

**preprocessing-sfa** is an Enduro preprocessing workflow for SFA SIPs.

- [Configuration](#configuration)
- [Local environment](#local-environment)
- [Makefile](#makefile)
- [Available activities](#available-activities)

## Configuration

The preprocessing workers need to share the filesystem with Enduro's a3m or
Archivematica workers. They must be connected to the same Temporal server
and related to each other with the namespace, task queue and workflow name.

### Preprocessing

The required configuration for the preprocessing worker:

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

Optional BagIt bag configuration:

```toml
[bagit]
checksumAlgorithm = "md5"
```

### Enduro

The preprocessing section for Enduro's configuration:

```toml
[preprocessing]
enabled = true
extract = true # Extract must be true for the preprocessing-sfa workflow.
sharedPath = "/home/enduro/preprocessing"

[preprocessing.temporal]
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"

# Enable the AIS poststorage workflow.
[[poststorage]]
namespace = "default"
taskQueue = "ais"
workflowName = "ais"
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

While we run the services inside a Kubernetes cluster we recomend installing
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
documentation to learn more about it.

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

In the Tilt UI header there is a cloud icon/button that can trigger the
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

## Available activities

* [Calculate SIP checksum](#calculate-sip-checksum)
* [Check for duplicate SIP](#check-for-duplicate-sip)
* [Unbag SIP](#unbag-sip)
* [Identify SIP structure](#identify-sip-structure)
* [Validate SIP structure](#validate-sip-structure)
* [Validate SIP name](#validate-sip-name)
* [Verify SIP manifest](#verify-sip-manifest)
* [Verify SIP checksums](#verify-sip-checksums)
* [Validate SIP files](#validate-sip-files)
* [Validate logical metadata](#validate-logical-metadata)
* [Create premis.xml](#create-premisxml)
* [Restrucuture SIP](#restructure-sip)
* [Create identifiers.json](#create-identifiersjson)
* [Other activities](#other-activities)

### Calculate SIP checksum

Part 1 of a 2-part activity around duplicate checking - see also:

* [Check for duplicate SIP](#check-for-duplicate-sip)

Generates and stores a checksum for the entire SIP, so it can be used to check
for duplicates

#### Steps

* Generate a SHA256 checksum for the incoming package
* Read SIP name
* Store SIP name and checksum in the persistence layer (`sips` table)

#### Success critera

* A SHA256 checksum is successfully generate for the SIP
* the SIP name and generated checksum are stored in the persistence layer

### Check for duplicate SIP

Part 2 of a 2-part activity around duplicate checking - see also:

* [Calculate SIP checksum](#calculate-sip-checksum)

Determines if an identical SIP  has previously been ingested

#### Steps

* Use the generated checksum from [part 1](#calculate-sip-checksum) to search
  for an existing match in the `sips` database table
* If an existing match is found, return a content error fro a duplicateSIP and
  terminate the workflow
* Else, continue to next activity

#### Success critera

* The activity is able to read the generated checksum and the `sips` database
  table
* No matching checksum is found the SIPs database table

### Unbag SIP

Extracts the contents of the bag.

Only runs if the SIP is a BagIt bag. If the SIP is not a bag, this activity will
not run.

#### Steps

* Check if SIP is a bag
* If yes, extract the contents of the bag for additional ingest processing
* Else, skip

#### Success critera

* Bag is successfully extracted

### Identify SIP structure

Determines the SIP type by analyzing the name and distinguishing features of
the package, based on eCH-0160 requirements and other internal policies.

Package types include:

* BornDigitalSIP
* DigitizedSIP
* BornDigitalAIP
* DigitizedAIP

#### Steps

* Base type is BornDigitalSIP; assume this is the SIP type unless other
  conditions are met
* Check if the package contains a `Prozess_Digitalisierung_PREMIS.xml` file
  * If yes, it is a Digitized package - either DigitizedSIP or DigitizedAIP
* Check if the package contains an additional directory
  * If yes, it is a migration AIP - either BornDigitalAIP or DigitizedAIP
* Compare check results and determine package type

#### Success criteria

* Package is successfully identified as one of the 4 supported types

### Validate SIP structure

Ensures that the SIP directory structure conforms to eCH-0160 specifications,
that no empty directories are included, and that there are no disallowed
characters used in file and directory names.

**Note**: Chracacter restrictions for file and directory names are based on some
of the requirements of the tools used by
[Archivematica](https://www.archivematica.org) during preservation processing -
at present, the file name cleanup steps in Archivematica cannot be modified or
disabled without forking. To ensure that SFA package metadata matches the
content, this validation check ensures that no disallowed characters are
included in file or directory names that might be automatically changed once
received by Archivematica.

#### Steps

* Read SIP type from previous activity
* Check for presence of `content` and `header` directories
* Check all file and directory names for invalid characters
* Check for empty directories

#### Success critera

* Files and directories only contain valid characters
  * `A-Z`, `a-z`, `0-9`, or `-_.()`
* SIPs contain `content` and `header` directories
  * If content type is an AIP, it also contains an `additional` directory
* No empty directories are found

### Validate SIP name

Ensure that submitted SIPs use the required naming convention for the identified
package type.

#### Steps

* Read SIP type from previous activity
* Use regular expression to validate SIP name based on identified type

#### Success critera

* SIP follows expected naming convention for package type:
  * BornDigitalSIP: `SIP_[YYYYMMDD]_[delivering office]_[reference]`
  * DigitizedSIP: `SIP_[YYYYMMDD]_Vecteur_[delivering office]_[reference]`

### Verify SIP manifest

Checks if all files and directories listed in the metadata manifest match those
found in the SIP, and that no extra files or directories are found.

#### Steps

* Load SIP metadata manifest into memory
* Parse the manifest contents and return a list of files and directories
* Parse the SIP and return a list of files and directories
* Compare lists
* Return a list of any missing files found in the manifest but not the SIP
* Return a list of unexpected files found in the SIP but not the manifest

#### Success critera

* There is a matching file or directory for every entry found in the
  `metadata.xml` (or `UpdatedAreldaMetadata.xml`) manifest
* No unexpected files that are not listed in the manifest are found

### Verify SIP checksums

Confirms that the checksums included in the metadata manifest match those
calculated during validation.

#### Steps

* Check if a given file exists in the manifest
* If yes, calculate a checksum - else skip
* Compare calculated checksum to manifest checksum

#### Success critera

* A checksum calculated using the same algorithm as the one used in the metadata
  file returns the same value as the one included in the metadata manifest for
  each file listed

### Validate SIP files

Ensures that files included in the SIP are well-formed and match their format
specifications.

#### Steps

* For PDFs, use [VeraPDF](https://github.com/veraPDF) to validate against the
  PDF specification
* Note: additional format validation checks will be added in the future

#### Success critera

* All files pass validation

### Validate logical metadata

Ensures that a logical metadata file is included for AIPs being migrated from
DIR and validates the file against a PREMIS schema file

**Note** : this activity uses some custom workflow code and a locally stored
copy of the PREMIS schema to run the general temporal activity
[xmlvalidate](https://github.com/artefactual-sdps/temporal-activities/blob/main/xmlvalidate/activity.go).

#### Steps

* Read package type from memory
* If package type is bornDigitalAIP or DigitizedAIP, check for XML file in
  `additional` directory
* If found, validate the XML file against a locally stored copy of the PREMIS
  schema; fail ingest if any errors are returned

#### Success critera

* Logical metadata file is found in the `additional` directory of the package
* Logical metadata file validates against PREMIS 3.x schema

### Create premis.xml

Generates a PREMIS XML file that captures ingest preservation actions performed
by Enduro as PREMIS events for inclusion in the resulting AIP METS file.

**NOTE**: This activity is broken up into 3 different activity files in
`/internal/activites`:

* `add_premis_agent.go`
* `add_premis_event.go`
* `add_premisobjects.go`

The XML output is then assembled via `/internal/premis/premis.go`.

#### Steps

* Review event details for all successful tasks
* Create premis.xml file in a new metadata directory
* Write PREMIS objects to file
* Write PREMIS events to file
* Write PREMIS agents to file

#### Success critera

* A `premis.xml` file is successfully generated with ingest events

### Restructure SIP

Reorganizes SIP directory structure into a Preservation Information Package
(PIP) that the preservation engine (Archivematica) can process.

#### Steps

* Check if `metadata` directory exists, else create a new `metadata` directory
* Move the `Prozess_Digitalisierung_PREMIS.xml` file to the `metadata` directory
* For AIPs, move the `UpdatedAreldaMetatdata.xml` and logical metadata files to
  the `metadata` directory
* Create an `objects` directory, and in that directory create a sub-directory
  with the SIP name
* Delete `xsd` directory and its contents from `header` directory
* Move `content` directory into the new `objects` directory
* Create a new `header` directory in objects
* Move the `metadata.xml` file into the new `header` directory
* Delete original top-level directories

#### Success critera

* XSD files are removed
* Restructured package now has `objects` and `metadata` directories immediately
  inside parent container
* All content for preservation is within the `objects` directory
* Enduro-generated PREMIS file is in the `metadata` directory
* For Digitized packages, `Prozess_Digitalisierung_PREMIS.xml` file is in the
  metadata directory

### Create identifiers.json

Extract original UUIDs from the SIP metadata file and add them to an
`identifiers.json` file added to the `metadata` directory of the package for
parsing by the preservation engine

#### Steps

* Parse SIP metadata file
* Extract persistent identifiers and write to memory
* Convert manifest file paths to the restructured PIP file paths
* Exclude any files in the manifest that aren't found in the PIP
* Using extracted identifiers, generate an `identifiers.json` file that conforms
  to Archivematica's expectations
* Move generated file to package `metadata` directory

#### Success critera

* An `identifiers.json` file is added to the `metadata` directory of the package
* UUIDs present in the original SIP metadata are maintained and used by the
  preservation engine during preservation processing

### Other activities

The SFA workflow that invokes the activities listed above (see the
[preprocessing.go](https://github.com/artefactual-sdps/preprocessing-sfa/blob/main/internal/workflow/preprocessing.go) 
file) also uses a number of other more general Enduro
[temporal activites](https://github.com/artefactual-sdps/temporal-activities), including:

* `archiveextract`
* `bagcreate`
* `bagvalidate`
* `ffvalidate`
* `xmlvalidate`

There is also one custom post-preservation workflow activity maintained in this
repository as well:

#### Create AIS metadata bundle

Extracts all relevant metadata from the SIP and resulting AIP and delivers it to
the AIS for synchronization.

##### Steps

* Generate a new XML document that combines the contents of the two source files
  (the SIP `metadata.xml` or `UpdatedAreldaMetadata.xml` file, and the AIP METS
  file)
* ZIP the generated file and deposit it in an `ais` MinIO bucket

##### Success criteria

* Metadata bundle is successfully generated and deposited
* AIS is able to receive and ingest the metadata


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
