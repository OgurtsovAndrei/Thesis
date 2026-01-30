# Reproducing MMPH Construction Failures

## Summary
MMPH construction fails for certain key sets with the error:
```
failed to build working approximate z-fast trie after 100 attempts, try to increase S and/or I
```

The validation always fails with the SAME number of keys (256 = one full bucket), regardless of the random seed used in retries.

## Saved Test Cases

Multiple failing test cases have been saved to JSON files in this directory:
- `failing_case_seed_*.json`

Each file contains:
- The seed used to generate the keys
- The error message
- The number of keys
- Raw byte data for each key
- Bit sizes for each key

## How to Debug

### Option 1: Use the debug test

1. Edit `debug_failing_case_test.go`
2. Uncomment `t.Skip()`
3. Update the filename to one of the saved cases
4. Run: `go test -v -run TestDebugFailingCase`

### Option 2: Load programmatically

```go
keys, err := rloc.LoadFailingCase("failing_case_seed_1769679785573021000.json")
if err != nil {
    panic(err)
}

// Now you can test with these exact keys
zt := zfasttrie.Build(keys)
rl, err := rloc.NewRangeLocator(zt)
// This should fail consistently
```

## Key Findings

1. **Always 256 keys fail** - This is exactly one bucket (bucket size = 256)
2. **Same keys fail across retries** - Changing the random seed doesn't help
3. **All delimiters are matched** - The trie recognizes all delimiter nodes correctly
4. **LowerBound returns wrong candidates** - For 256 keys, the 3 candidates returned by ApproxZFastTrie.LowerBound() don't include the correct delimiter

## Hypothesis

The bug is likely in how `minChild`, `minGreaterChild`, or `parent` relationships are computed in the ApproxZFastTrie for certain key patterns, especially with:
- Mixed-size strings
- TrieCompare ordering
- Specific prefix patterns

The issue is deterministic (not a randomness problem) and affects a specific structural pattern in the trie.

## Type Parameters Used

Current failing configuration:
- E (extent length): uint16
- S (signature): uint16
- I (delimiter index): uint16

The error message suggests increasing S and/or I, but this just masks the underlying structural bug.
