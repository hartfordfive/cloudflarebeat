# Cloudflarebeat

Welcome to Cloudflarebeat.

Ensure that this folder is at the following location:
`${GOPATH}/github.com/hartfordfive`

## Getting Started with Cloudflarebeat

### Requirements

* [Golang](https://golang.org/dl/) 1.7

### Init Project
To get running with Cloudflarebeat and also install the
dependencies, run the following command:

```
make setup
```

It will create a clean git history for each major step. Note that you can always rewrite the history if you wish before pushing your changes.

To push Cloudflarebeat in the git repository, run the following commands:

```
git remote set-url origin https://github.com/hartfordfive/cloudflarebeat
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for Cloudflarebeat run the command below. This will generate a binary
in the same directory with the name cloudflarebeat.

```
make
```


### Run

To run Cloudflarebeat with debugging output enabled, run:

```
./cloudflarebeat -c cloudflarebeat.yml -e -d "*"
```

For details of command line options, view the following links:

- https://www.elastic.co/guide/en/beats/libbeat/master/config-file-format-cli.html
- https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-command-line.html


## Cloudflarebeat specific configuration options

- `cloudflarebeat.period` : The period at which the cloudflare logs will be fetched.  (Default value is 1800s/30mins which is the default suggested by the Enterprise Log Share API documentation page.)
- `cloudflarebeat.api_key` : The API key of the user account (mandatory)
- `cloudflarebeat.email` : The email address of the user account (mandatory)
- `cloudflarebeat.zone_tag` : The zone tag of the domain for which you want to access the enterpise logs (mandatory)
- `cloudflarebeat.state_file_storage_type` : The type of storage for the state file, either `disk`, `s3`, or `consul`, which keeps track of the current progress. (Defau)
- `cloudflarebeat.aws_access_key` : The user AWS access key, if S3 storage selected.
- `cloudflarebeat.aws_secret_access_key` : The user AWS secret access key, if S3 storage selected.

## Filtering out specific logs and/or log properties


### Test

To test Cloudflarebeat, run the following command:

```
make testsuite
```

alternatively:
```
make unit-tests
make system-tests
make integration-tests
make coverage-report
```

The test coverage is reported in the folder `./build/coverage/`

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `etc/fields.yml`.
To generate etc/cloudflarebeat.template.json and etc/cloudflarebeat.asciidoc

```
make update
```


### Cleanup

To clean  Cloudflarebeat source code, run the following commands:

```
make fmt
make simplify
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone Cloudflarebeat from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/github.com/hartfordfive
cd ${GOPATH}/github.com/hartfordfive
git clone https://github.com/hartfordfive/cloudflarebeat
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make package
```

This will fetch and create all images required for the build process. The hole process to finish can take several minutes.
