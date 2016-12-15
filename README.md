# Cloudflarebeat

Custom beat to fetch Cloudflare logs via the Enterprise Log Share API.

Ensure that this folder is at the following location:
`${GOPATH}/github.com/hartfordfive`


## Disclaimer

Cloudflarebeat is currently in beta.  It may have bugs and likely various optimizations that can be made.
If you find any of these, please create an issue or even a pull request if you're familiar with development for beats library.

## Acknoledgements

Special thank you to [Lightspeed POS](http://www.lightspeedhq.com) for providing access to test data, feedback and suggestions.

## Getting Started with Cloudflarebeat

### Basic Overview of Application Design

1. API request is made to the Cloudflare ELS endpoint for logs within a specific time range, ending at most 30 minutes AGO
2. When the response is received, the gzip content is saved into a local file.
3. Individual JSON log entries are read from the file one by one, individual fields are added into the appropriate struct, and then sent off to be published.
4. Once all log entries in the file have been processed, the remaining log file is deleted, unless the user has specified the option to keep log original files.


### Requirements

* [Golang](https://golang.org/dl/) 1.7
* [goreq](https://github.com/franela/goreq)
* [ffjson](https://github.com/pquerna/ffjson/ffjson)


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

- `cloudflarebeat.period` : The period at which the cloudflare logs will be fetched.  Regardless of the period, logs are always fetched from **30 MINUTES AGO - PERIOD** to **NOW**. (Default value of period is 1800s/30mins)  
- `cloudflarebeat.api_key` : The API key of the user account (mandatory)
- `cloudflarebeat.email` : The email address of the user account (mandatory)
- `cloudflarebeat.zone_tag` : The zone tag of the domain for which you want to access the enterpise logs (mandatory)
- `cloudflarebeat.state_file_path` : The path in which the state file will be saved (applicable only with `disk` storage type)
- `cloudflarebeat.state_file_name` : The name of the state file
- `cloudflarebeat.state_file_storage_type` : The type of storage for the state file, either `disk` or `s3`, which keeps track of the current progress. (Default: disk)
- `cloudflarebeat.aws_access_key` : The user AWS access key, if S3 storage selected.
- `cloudflarebeat.aws_secret_access_key` : The user AWS secret access key, if S3 storage selected.

## Using S3 Storage for state file

For cloudflarebeat, it's probably best to create a seperate IAM user account, without a password and only this sample policy file.  Best to limit the access of your user as a security practice.

Below is a sample of what the policy file would look like for the S3 storage.  Please note you should replace `my-cloudflarebeat-bucket-name` with your bucket name that you've created in S3.

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::my-cloudflarebeat-bucket-name"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:GetObject",
                "s3:DeleteObject"
            ],
            "Resource": [
                "arn:aws:s3:::my-cloudflarebeat-bucket-name/*"
            ]
        }
    ]
}
```

## Filtering out specific logs and/or log properties

Please read the beats [documentation regarding processors](https://www.elastic.co/guide/en/beats/filebeat/master/configuration-processors.html).  This will allow you to filter events by field values or even remove event fields.

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


## Author

Alain Lefebvre <hartfordfive 'at' gmail.com>

## License

Covered under the Apache License, Version 2.0
Copyright (c) 2016 Alain Lefebvre