# Testability Recommendations

1. **Avoid `log.Fatalf` in Logic**: `argMassage` currently uses `log.Fatalf` when an invalid IP is provided. This terminates the entire test process. Returning an `error` instead would allow for unit testing of invalid inputs and edge cases.

2. **Decouple I/O from Logic**: `ipcmd` is currently tied to the filesystem because it opens files internally using `os.Open`. Passing in a slice of `io.Reader` (or a more abstract `Input` struct) instead of filenames would make it easier to test without needing to create temporary files on disk.
   - **Example**: Use an abstraction for inputs:
     ```go
     type Input struct {
         Name   string
         Reader io.Reader
     }
     ```
   - **Benefits**: This allows tests to pass `strings.NewReader("...")` to simulate file contents entirely in memory, avoiding the overhead and fragility of temporary files.

3. **Refactor Match Logic into Pure Functions**: Much of the logic inside `ipcmd` (the nested loops and switch statements for matching) could be extracted into a smaller, pure function (e.g., `func matches(target *ipaddr.IPAddress, candidate *ipaddr.IPAddress, mode string) bool`). This would enable faster and more granular unit tests of the IP matching logic without any I/O overhead.

4. **Injectable Stdin**: The current implementation checks `IsStdin` and then uses `os.Stdin` directly, which makes testing the stdin path difficult. Accepting an `io.Reader` for "stdin" instead of hardcoding `os.Stdin` would allow for easy testing of this code path.
   - **Example**: Add a `Stdin io.Reader` field to `cliArgStruct`. In `main()`, set it to `os.Stdin`.
   - **Benefits**: Tests can provide a custom reader (like `bytes.NewBuffer`) to simulate user input and verify the stdin processing path without manual intervention or terminal blocking.
