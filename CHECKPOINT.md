# CHECKPOINT — ipfind

Date: 2026-02-12

Summary
-------
- Implemented `ipfind` CLI in Go with tests and a `Justfile`.
- Added a `goreleaser` config (now v2) and a `goreleaser-setup` branch for release automation work.
- Created a manual Homebrew formula at `homebrew-tap/Formula/ipfind.rb` and updated its `sha256` to a computed value.

Focus areas and recommended next steps
------------------------------------

1) Homebrew (safe/manual first)
  - We currently keep the single-file formula at `homebrew-tap/Formula/ipfind.rb` in the repo.
  - That formula is manual: you (or CI) must update `sha256` to match the final release archive before publishing the tap.
  - Recommended short-term action: create the actual GitHub tap repository (e.g., `ewosborne/homebrew-tap`) and push the `homebrew-tap/Formula/ipfind.rb` file there when you do your first release.

2) Automating Homebrew with goreleaser (safe hybrid)
  - goreleaser v2 can generate and push Homebrew formulae for you, but it requires:
    - A `brews` block in `.goreleaser.yml` using the v2 schema.
    - A `GITHUB_TOKEN` (or similar) with push rights to the tap repo in CI.
  - We added a non-active example under `.goreleaser.yml:homebrew_example` to show v2 schema without enabling it.
  - To enable automation safely:
    1. Create the tap repo and give the release runner push rights.
    2. Replace the `homebrew_example` with a top-level `brews` block in `.goreleaser.yml`.
    3. Test with dry-runs in CI and locally using `goreleaser release --snapshot --skip publish --clean --skip=homebrew`.
    4. When satisfied, run a guarded release that omits `--skip=homebrew` (CI only) to let goreleaser publish the formula.

3) goreleaser configuration and CI
  - Current state: `.goreleaser.yml` is upgraded to `version: 2` and the repo contains a GitHub Actions workflow template.
  - Next steps:
    - Update `.github/workflows/goreleaser.yml` to ensure it runs with a token that has the minimum required permissions (push to tap only when you want it).
    - Keep `--skip` flags in CI for early tests. Use snapshots for local verification.

4) Release process notes
  - Always compute and verify SHA256 for the release source tarball that will be published by GitHub (goreleaser can compute and insert it when publishing the formula).
  - Tagging policy: create signed or annotated tags (e.g., `v0.1.0`) for releases so goreleaser picks the right archive.

Decisions made here
-------------------
- We opted for a safe hybrid approach: manual formula in `homebrew-tap/Formula/ipfind.rb` and a documented, non-active v2 example in `.goreleaser.yml`.
- This gives you manual control now and an easy migration path to automated Homebrew publishing later.

What I pushed in this checkpoint
--------------------------------
- `homebrew-tap/Formula/ipfind.rb` (sha256 updated)
- `.goreleaser.yml` (version: 2 + `homebrew_example` block)
- `docs/RELEASE.md` (safe hybrid workflow documented)

Suggested immediate next move (when ready)
-----------------------------------------
1. Create the GitHub tap repo and add the formula there manually for the first release.
2. Configure CI `GITHUB_TOKEN` with push rights to the tap when you want to enable automation.
3. Convert `homebrew_example` → `brews` and test with `--skip=homebrew` before letting goreleaser push.

If you want, I can do steps 2–3 for you next time (convert the block and run guarded CI dry-runs).

Future features to consider
---------------------------
- IPv6 support: add detection and parsing of IPv6 addresses and prefixes alongside IPv4. Decide whether flags/modes apply consistently for v6 (e.g., `--exact` should match exact IPv6 addresses; `--subnet` should accept IPv6 prefixes).
- Multicast support (v4 and v6): include optional filters for multicast ranges (e.g., 224.0.0.0/4 for IPv4, ff00::/8 for IPv6) or a flag like `--multicast-only` to surface multicast addresses.
- don't need to read the entire stdin unless it's -l.  otherwise can do this line by line.
- add an option to pass in a net+mask and get all matching or smaller subnets.

Testing improvements
--------------------
- Add unit tests and integration tests for IPv6 addresses and prefixes, and combinations of mixed IPv4/IPv6 input.
- Add tests for multicast detection (both IPv4 and IPv6 multicast ranges).
- Add tests for invalid input: malformed IPs, malformed CIDRs, out-of-range octets, bad mask ranges, and invalid flag combinations. Ensure the CLI exits with non-zero codes and emits helpful error messages for these cases.

Notes
-----
- When adding IPv6 support, review parsing and matching logic to use `net/netip`'s `Addr` and `Prefix` types for both families and ensure that mask-range semantics make sense for v6 (masks are larger; consider limiting the practical mask-range defaults).
- Multicast and IPv6 support will slightly change the CLI output and test expectations; plan to add sample files with IPv6/multicast examples for automated tests.
