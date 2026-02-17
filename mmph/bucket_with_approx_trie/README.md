# MMPH Module Notes

Key observations about `bucket_with_approx_trie` with direct links to the detailed sources.

- **Correctness via Las Vegas Validation:** The `ApproxZFastTrie` used for bucket identification is probabilistic and can yield both False Positives and False Negatives (see [FP & FN in AZFT](../../trie/azft/FP%26FN%20in%20AZFT.md)). To ensure 100% correctness for the input key set, the builder:
    1. Builds the trie on bucket delimiters.
    2. Validates the trie by checking every input key resolves to the correct bucket.
    3. Retries with a new seed if validation fails (up to 100 times).
  This converts the Monte Carlo error probability of the trie into a construction-time overhead. See `buildValidatedTrieWithIndices` in `hash.go` and the theoretical analysis in `analysis_summary.md`.
- The bucket size is effectively **fixed to 256**, even though the code comments reference `b = log n`; the `minBucketSize` computed from `log2(n)` is not used. See `bucket_with_approx_trie/hash.go`.
- Correctness relies on **sorted input keys**; the builder does not sort and assumes sorted order for two-pointer validation. See `bucket_with_approx_trie/hash.go` and tests that explicitly sort in `bucket_with_approx_trie/prop_test.go`.
- The MPHF is built on **key hashes**; hash collisions or duplicate keys within a bucket can break rank reconstruction, and there is no explicit uniqueness check. See `bucket_with_approx_trie/hash.go`.
- The trie lookup uses a **length-consistency collision filter** (`extentLen < fFast`) in `GetExistingPrefix`, which removes a class of MPH-induced false positives not modeled in the paper. See `zfasttrie/getexistingprefix_collision_filter.md` and `zfasttrie/approx_z_fast_trie.go`.
