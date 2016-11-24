## Changelog

beta-0.1.0
-----
* First initial beta release.

beta-0.2.0
-----
* Added S3 state file storage option ([Issue #2](https://github.com/hartfordfive/cloudflarebeat/issues/4))
* Fixed bug where nanosecond timestamp fields were not always received in scientific notiation, thus causing an error with the interface type conversion ([Iusse #4](https://github.com/hartfordfive/cloudflarebeat/issues/4))
* Logs are now downloaded and stored in a gzip file, and then read sequentially in order to reduce memory requirement, which was previously higher due to all in-memory download and processing. 
* Added configuration options `state_file_path` and `state_file_name` to allow custimization of state file name and file path.
