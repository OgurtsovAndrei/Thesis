# RocksDB Filters

## Overview

RocksDB implements a sophisticated multi-layer filtering system to optimize query performance by reducing unnecessary I/O operations. This report summarizes the 9 primary filtering mechanisms, their solved problems, and key implementation techniques.

## Filter Types Summary

### 1. **SST Metadata Filtering** (Always Active)
- **Problem Solved**: Eliminate reading entire SST files that cannot possibly contain relevant keys for range queries
- **Key Idea**: Store min/max key bounds in SST metadata and use range overlap detection to skip non-overlapping files
- **Impact**: 85-99% I/O reduction for range queries, forms foundation for all other filtering

### 2. **Bloom Filters** (Probabilistic Point Filtering)
- **Problem Solved**: Expensive disk I/O for non-existent key lookups in LSM-tree structures
- **Key Idea**: Probabilistic data structure with no false negatives - if filter says "not present", key is definitely absent
- **Impact**: 90-99% I/O reduction for point queries on missing keys, ~1% false positive rate typical

### 3. **Block-Level Filtering** (Hierarchical Memory Management)
- **Problem Solved**: Memory pressure from large monolithic filters and cache inefficiency for partial access
- **Key Idea**: Partition filters into smaller blocks with selective loading based on query needs
- **Impact**: 30% memory reduction with partitioned filters, improved cache utilization

### 4. **Prefix Extractors** (Structured Key Optimization)
- **Problem Solved**: Poor Bloom filter utilization for structured keys and inefficient range queries on composite keys
- **Key Idea**: Extract meaningful prefixes from keys and store them in Bloom filters, to enable prefix-based Bloom filtering and range optimizations.
- **Impact**: 80-95% speedup for prefix-based range queries, massive memory savings when many keys share prefixes

### 5. **Ribbon Filters** (Space-Efficient Alternative)
- **Problem Solved**: Memory overhead of traditional Bloom filters (44% above theoretical optimum)
- **Key Idea**: XOR-based perfect hash static functions achieving near-theoretical space efficiency
- **Impact**: 27-30% space savings vs Bloom filters, trade CPU cost for memory efficiency

### 6. **Compaction Filters** (Data Lifecycle Management)
- **Problem Solved**: Need for custom data expiration, transformation, and selective deletion during normal operations
- **Key Idea**: Callback interface during compaction to filter, transform, or skip key ranges inline with background processing
- **Impact**: Zero additional I/O for data lifecycle management, 95-99% reduction for expired data cleanup

### 7. **SST Query Filters** (Experimental Range Filtering)
- **Problem Solved**: Limited range filtering granularity between prefix-level and metadata-level filtering
- **Key Idea**: Store min/max bounds for extracted key segments to enable fine-grained range filtering on composite keys
- **Impact**: 3x speedup for high-selectivity structured key queries, segment-based optimization

### 8. **Table Filters** (File-Level Selection)
- **Problem Solved**: Need to selectively exclude entire SST files based on custom criteria or application logic
- **Key Idea**: Callback mechanism using table properties to filter files before any I/O operations
- **Impact**: 50-90% file access reduction for selective queries, zero I/O overhead

### 9. **Timestamp Filtering** (Temporal Range Optimization)
- **Problem Solved**: Efficient querying of time-series data and multi-version concurrency control
- **Key Idea**: Built-in timestamp support in key encoding with temporal bound filtering during iteration
- **Impact**: 90-99% data reduction for time-range queries, enables efficient MVCC implementations

## Filtering Hierarchy and Performance

```
Query Processing Flow:
┌─ Table Filters (File Selection) ──────────────────┐
│  ├─ SST Metadata Filtering (Range Bounds)         │
│  │  ├─ SST Query Filters (Segment Bounds)         │
│  │  │  ├─ Block-Level Filtering (Memory)          │
│  │  │  │  ├─ Bloom/Ribbon Filters (Point)         │
│  │  │  │  └─ Prefix Extractors (Range)            │
│  │  │  └─ Timestamp Filtering (Temporal)          │
│  │  └─ Compaction Filters (Lifecycle)             │
│  └─ Data Processing                               │
└───────────────────────────────────────────────────┘
```

## Key Performance Characteristics

### Memory Usage (per SST file)
- **SST Metadata**: ~250 bytes + 2×key_length
- **Bloom Filter**: keys × bits_per_key ÷ 8 (typically ~1-2MB per million keys)
- **Ribbon Filter**: 70% of Bloom memory usage
- **Compaction Filter**: Negligible runtime overhead
- **Other filters**: < 1KB overhead each

### Query Time Impact
- **Metadata filtering**: O(log files_per_level) per level
- **Bloom/Ribbon lookup**: O(1) with k hash functions
- **Prefix extraction**: O(1) for fixed-width, O(key_length) for delimited
- **Filter evaluation**: 100ns - 1μs per filter check

### Construction Overhead
- **Bloom construction**: ~32 ns/key
- **Ribbon construction**: ~140 ns/key (4.4x slower than Bloom)
- **All others**: Minimal impact on write path
