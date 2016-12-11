## Changelog

beta-0.1.0
-----
* First initial beta release.

beta-0.2.0
-----
* Added S3 state file storage option ([Issue #2](https://github.com/hartfordfive/cloudflarebeat/issues/4))
* Fixed bug where nanosecond timestamp fields were not always received in scientific notiation, thus causing an error with the interface type conversion ([Iusse #4](https://github.com/hartfordfive/cloudflarebeat/issues/4))
* Added `BuildMapStr` function which builds the final event to be sent.  This ensures that fields with types such as `ip` will simply be ommitted if they are an empty string, otherwise this used to cause a mapping exception.
* Added configuration options `state_file_path` and `state_file_name` to allow custimization of state file name and file path.
* Logs are now fetched/processed/published within seperate goroutines.  This allows for an overall faster opperation.
   * A first goroutine splits the specified period in individual segments, and each downloaded into a gzip file
   * A second goroutine as gzip log files are ready, they are read and each log entry is converted to an appopriate `MapStr` struct and push to a channel for final publishing
   * The third goroutine reads events from pending channel and publishes them via the selected output
* Added `delete_logfile_after_processing` option, which is set to true by default.  Only set this to false if you need to keep the gzip log files.
* Included the `zone_tag` in the state file name so that each Cloudflare zone will have its own state file.
