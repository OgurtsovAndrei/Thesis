# MMPH Module Notes

Key observations about `bucket_with_approx_trie` with direct links to the detailed sources.

- The build validates the approximate z-fast trie by checking **all keys** and retries up to 100 seeds; correctness depends on this validation pass. See the implementation in `bucket_with_approx_trie/hash.go` and the theory/empirical discussion in `bucket_with_approx_trie/analysis_summary.md`.
- The bucket size is effectively **fixed to 256**, even though the code comments reference `b = log n`; the `minBucketSize` computed from `log2(n)` is not used. See `bucket_with_approx_trie/hash.go`.
- Correctness relies on **sorted input keys**; the builder does not sort and assumes sorted order for two-pointer validation. See `bucket_with_approx_trie/hash.go` and tests that explicitly sort in `bucket_with_approx_trie/prop_test.go`.
- The MPHF is built on **key hashes**; hash collisions or duplicate keys within a bucket can break rank reconstruction, and there is no explicit uniqueness check. See `bucket_with_approx_trie/hash.go`.
- The trie lookup uses a **length-consistency collision filter** (`extentLen < fFast`) in `GetExistingPrefix`, which removes a class of MPH-induced false positives not modeled in the paper. See `zfasttrie/getexistingprefix_collision_filter.md` and `zfasttrie/approx_z_fast_trie.go`.
