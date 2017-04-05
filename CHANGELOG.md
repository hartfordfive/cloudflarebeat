## Changelog

beta-0.1.0
-----
* First initial beta release.

beta-0.2.0
-----
* Added S3 state file storage option ([Issue #2](https://github.com/hartfordfive/cloudflarebeat/issues/2))
* Fixed bug where nanosecond timestamp fields were not always received in scientific notiation, thus causing an error with the interface type conversion ([Iusse #4](https://github.com/hartfordfive/cloudflarebeat/issues/4))
* Logs are now downloaded and stored in a gzip file, and then read sequentially in order to reduce memory requirement, which was previously higher due to all in-memory download and processing. ([Issue #9](https://github.com/hartfordfive/cloudflarebeat/issues/9))
* Added new configuration options `state_file_path`, `state_file_name`, `delete_logfile_after_processing`, `processed_events_buffer_size`
* Updated logic so that logs are downloaded immediately, without delay in the case where the process has stopped and the time elapsed is greater than the configured period.
* Fixed Elasticsearch 5.x index template.
* Explicitly closed log files handles upon completion as the *too many files open* error was begining to occur after the process was running for over a few days ([Issue #11](https://github.com/hartfordfive/cloudflarebeat/issues/11))
* Added `BuildMapStr` function which builds the final event to be sent.  This ensures that fields with types such as `ip` will simply be ommitted if they are an empty string, otherwise this used to cause a mapping exception.
* Included the `zone_tag` in the state file name so that each Cloudflare zone will have its own state file.


beta-0.3.0
----
* Added ability to read the **.tar.gz** log files from a specified directory and exit after completion ([Issue #10](https://github.com/hartfordfive/cloudflarebeat/issues/10))
* Fixed issue where the process hangs when a log file segment download fails, the file is never sent to the `LogFilesReady` channel, therefore never gets processord or acked in the wait group. ([Issue #15](https://github.com/hartfordfive/cloudflarebeat/issues/15))
* Added ability to specify the path, via the config `tmp_logs_dir` option, where downloaded cloudflare logs are saved to. (Default is `"tmp_logs/"`)
* Added option to process log file segments in parallel via the `parallel_log_processing` config option.  If set to true (default is **false**), as log file segment downloads are complete, they are processed in a seperate goroutine instead of processing each segment sequentially.