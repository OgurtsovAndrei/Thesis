package ere_pef

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"
	"testing"
	"path/filepath"
)

func loadSOSDUint64(path string, maxKeys int) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	readN := int(count)
	if maxKeys > 0 && maxKeys < readN {
		readN = maxKeys
	}

	keys := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, &keys); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := keys[:0]
	for i, k := range keys {
		if i == 0 || k != keys[i-1] {
			out = append(out, k)
		}
	}
	return out, nil
}

func loadSOSDUint32As64(path string, maxKeys int) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	readN := int(count)
	if maxKeys > 0 && maxKeys < readN {
		readN = maxKeys
	}

	keys32 := make([]uint32, readN)
	if err := binary.Read(f, binary.LittleEndian, &keys32); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}

	keys := make([]uint64, readN)
	for i, k := range keys32 {
		keys[i] = uint64(k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := keys[:0]
	for i, k := range keys {
		if i == 0 || k != keys[i-1] {
			out = append(out, k)
		}
	}
	return out, nil
}

func analyzePEF(name string, p *PEF) {
	fmt.Printf("========================================================\n")
	fmt.Printf("SOSD Research Report: Dataset [%s]\n", name)
	fmt.Printf("Total Keys (n): %d\n", p.n)
	fmt.Printf("Number of Chunks: %d\n", len(p.chunks))
	fmt.Printf("Shared RSDic size: %d bits\n", p.rs.Num())
	fmt.Printf("Shared LowBits size: %d bits\n", p.lowBitsN)
	fmt.Printf("Total BPK (Bits Per Key): %.2f\n", float64(p.SizeInBits())/float64(p.n))
	fmt.Println()

	var countAllOnes, countEF, countBitmap int
	var keysAllOnes, keysEF, keysBitmap uint64
	var totalBuckets uint64
	var maxEll uint64
	var totalEll uint64

	bucketHist := make(map[uint64]int)

	for i, c := range p.chunks {
		kind := c.kind()
		n := uint64(c.n())

		switch kind {
		case kindAllOnes:
			countAllOnes++
			keysAllOnes += n
		case kindBitmap:
			countBitmap++
			keysBitmap += n
		case kindEF:
			countEF++
			keysEF += n
			
			base := uint64(0)
			if i > 0 {
				base = p.chunks[i-1].last + 1
			}
			lastRel := c.last - base
			
			var ell uint64
			if lastRel >= n {
				ell = uint64(bits.Len64(lastRel/n) - 1)
			}
			if ell > maxEll {
				maxEll = ell
			}
			totalEll += ell

			meta := p.efMeta[c.metaIdx]
			
			numBuckets := (lastRel >> ell) + 1
			totalBuckets += numBuckets
			
			for b := uint64(0); b < numBuckets; b++ {
				start := p.rs.Select1(meta.onesBefore+b) - meta.globalOff - b
				end := p.rs.Select1(meta.onesBefore+b+1) - meta.globalOff - (b + 1)
				bSize := end - start
				bucketHist[bSize]++
			}
		}
	}

	fmt.Printf("Chunk Types Distribution:\n")
	fmt.Printf(" - AllOnes : %7d chunks (%5.1f%% of keys)\n", countAllOnes, 100*float64(keysAllOnes)/float64(p.n))
	fmt.Printf(" - Bitmap  : %7d chunks (%5.1f%% of keys)\n", countBitmap, 100*float64(keysBitmap)/float64(p.n))
	fmt.Printf(" - EliasFano: %7d chunks (%5.1f%% of keys)\n", countEF, 100*float64(keysEF)/float64(p.n))
	fmt.Println()

	if countEF > 0 {
		fmt.Printf("Elias-Fano Internals:\n")
		fmt.Printf(" - Average Low bits (ell) : %.2f\n", float64(totalEll)/float64(countEF))
		fmt.Printf(" - Max Low bits (ell)     : %d\n", maxEll)
		fmt.Printf(" - Total Buckets          : %d\n", totalBuckets)
		fmt.Printf(" - Average Keys per Bucket: %.2f\n", float64(keysEF)/float64(totalBuckets))
		
		fmt.Printf(" - Bucket Size Histogram (Top 5):\n")
		var sizes []uint64
		for k := range bucketHist {
			sizes = append(sizes, k)
		}
		sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })
		for i, size := range sizes {
			if i >= 5 && i != len(sizes)-1 {
				if i == 5 {
					fmt.Printf("    ...\n")
				}
				continue
			}
			fmt.Printf("    [%2d keys]: %d buckets\n", size, bucketHist[size])
		}
	}
	fmt.Printf("========================================================\n\n")
}

func TestSOSDPEFStats(t *testing.T) {
	fmt.Println(strings.Repeat("\n", 2))
	
	baseDir := "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/sosd_data"

	// test on ~10M keys to be fast but representative
	maxKeys := 10_000_000

	datasets := []struct{
		path string
		is32 bool
		name string
	}{
		{"fb_200M_uint64", false, "SOSD Facebook (fb)"},
		{"wiki_ts_200M_uint64", false, "SOSD Wiki Timestamps (wiki)"},
		{"books_200M_uint32", true, "SOSD Books (books)"},
		{"osm_cellids_800M_uint64", false, "SOSD OpenStreetMap (osm)"},
	}

	for _, ds := range datasets {
		fullPath := filepath.Join(baseDir, ds.path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fmt.Printf("Skipping %s, file not found: %s\n", ds.name, fullPath)
			continue
		}

		var keys []uint64
		var err error
		if ds.is32 {
			keys, err = loadSOSDUint32As64(fullPath, maxKeys)
		} else {
			keys, err = loadSOSDUint64(fullPath, maxKeys)
		}
		
		if err != nil {
			t.Fatalf("Failed to load %s: %v", ds.name, err)
		}

		p, err := NewPEF(keys, 64)
		if err != nil {
			t.Fatalf("Failed to build PEF for %s: %v", ds.name, err)
		}
		analyzePEF(ds.name, p)
	}
}
