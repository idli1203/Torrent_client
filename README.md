Single-File Torrent Downloader (Leech-Only)

This is a minimalistic approach on building a torrent client which supports a single file download and supports only HTTPS trackers. I have currently only implemented the download feature
and there is no uploading of torrent files supported. This is a feature planned to be implemented along with directory based file download support.

To run this just download the file and enter the folder and run command : go run main.go input_file_path output_file_path

### V2 version of this project is in progress:

1. Fix syntax error (blocking)
2. Terminal UI (best user impact)
3. Resume downloads (essential for large files)
4. Speed optimizations (backlog, block size)
5. UDP tracker support
6. Multi-file support
7. Fix the ugly error handling
8. Fix the memory usage
9. LASTLY , do a pprof check to ensure best performance.
