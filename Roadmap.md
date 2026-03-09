## Plan

- [x] `SuccinctBitVector` Лаконичные Битовые Векторы
- [x] `LocalExactRangeLocator` 
    - [x] Weak Prefix Search `Hollow Z Fast Trie`
        - [x] Deterministic Z Fast Trie
        - [x] Probabilistic Z Fast Trie
        - [x] MMPH (Монтонное Минимальное Совершенное Хеширование)
        - [x] Encoded Handles table T (handles -> len(e_alpha))
    - [x] Range Locator
- [x] `ExactRangeEmptiness`
- [x] `ApproximateRangeEmptiness`

### `ApproximateRangeEmptiness` [approximate_range_emptiness.md](local_exact_range/approximate_range_emptiness.md)

Probabilistic data structure that answers range emptiness queries with a false positive probability $\epsilon$.
Based on Section 4 of *Approximate Range Emptiness in Constant Time and Optimal Space*.
Breaks the linear space dependence on $L$ by using hashed fingerprints.

#### Expected Complexity
- **Space**: $O(n \log(L_{interval}/\epsilon))$ bits. Approximately **15-20 bits/key** for $\epsilon=0.01$.
- **Time**: $O(1)$ constant time query.

#### API
- `func NewApproximateRangeEmptiness(keys []bits.BitString, epsilon float64, maxL uint32) (*ApproximateRangeEmptiness, error)`
    - `keys`: sorted slice of elements $S$.
    - `epsilon`: desired false positive probability.
    - `maxL`: maximum length of query interval to support optimally.
- `func (are *ApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool`
    - Returns `true` if $[a, b] \cap S = \emptyset$.
    - Returns `false` with high probability if $[a, b] \cap S \neq \emptyset$.

### `ExactRangeEmptiness` [exact_range_emptiness.md](local_exact_range/exact_range_emptiness.md)

Succinct data structure that answers exact 1D range emptiness queries $[a, b] \cap S \neq \emptyset$ in $O(1)$ time.
Based on Section 3 of *Approximate Range Emptiness in Constant Time and Optimal Space*.
Requires dividing the universe into blocks and utilizing `SuccinctBitVector` for summary structures.

#### API
- `func NewExactRangeEmptiness(keys []bits.BitString, universe bits.BitString) (*ExactRangeEmptiness, error)`
    - `keys`: sorted slice of elements $S$.
    - `universe`: Maximum possible value in the universe $U$ (to support `BitString`s larger than `uint64`).
    - Builds the block summary structures and succinct representations in $O(n)$ time.
- `func (ere *ExactRangeEmptiness) IsEmpty(a, b bits.BitString) bool`
    - Returns `true` if the interval $[a, b]$ contains NO elements from $S$.
    - Returns `false` if the interval contains at least one element.
    - Executes in $O(1)$ time (relative to word size / `BitString` operations).

### `SuccinctBitVector` Лаконичные Битовые Векторы [succinct_bit_vector.md](succinct_bit_vector/SuccinctBitVector.md)

Succinct Bit Vector — это пространственно-эффективная структура данных, позволяющая хранить битовый массив длины $N$,
занимая $N + o(N)$
памяти, и поддерживающая операции Rank и Select за время $O(1)$.
More details in [succinct_bit_vector.md](succinct_bit_vector/SuccinctBitVector.md)

#### Api

- `func NewSuccinctBitVector(data []uint64, n int) SuccinctBitVector`
    - Build `SuccinctBitVector` - $O(n)$
- `func (bv *SuccinctBitVector) Access(i int) bool`
    - Возвращает значение бита по индексу $i$ ($0$ или $1$). in $O(1)$
- `func (bv *SuccinctBitVector) Rank1(i int) int`
    - Возвращает количество установленных бит ($1$) в диапазоне $[0, i)$. in $O(1)$
- `func (bv *SuccinctBitVector) Select1(k int) int`
    - Возвращает индекс $k$-й установленной единицы в массиве. in $O(1)^*$

### Z Fast Trie [zft_theory.md](trie/zft/zft_theory.md)

#### Api

`interface ZFastTrie`

- `Build(keys []string)`
- `PrefixSearch(query string) PrefixSearchResult` - for example range [L, R] in Weak Prefix Search Task
- `PredecessorSearch(query string) PredecessorSearchResult` - for example to find bucket in MMPH

#### Deterministic Implementation [zft](trie/zft) was adopted from [ctriepp](https://gitlab.com/habatakitai/ctriepp)

### MMPH (Monotone Minimal Perfect Hashing) [mmph/README.md](mmph/README.md)

Биективное отображение элементов отсортированного множества ключей в их порядковые номера с сохранением
лексикографического порядка.

#### Два варианта реализации

**Вариант A: Time-optimized**

- Query: O(1), Space: O(n log w)
- Техника: Bucketing with LCP
- Применение: Range Locator

**Вариант B: Space-optimized**

- Query: O(log w), Space: O(n log log w)
- Техника: Relative ranking + probabilistic trie
- Применение: Memory-constrained environments

#### API

- `Build(sorted_keys []string)` - построение за O(n log w)
- `Rank(key string) -> int` - получение ранга ключа

## TODO:

Check:

- https://javadoc.io/doc/it.unimi.dsi/sux4j/latest/it/unimi/dsi/sux4j/mph/HollowTrieDistributor.html
- https://javadoc.io/doc/it.unimi.dsi/sux4j/latest/it/unimi/dsi/sux4j/mph/HollowTrieDistributorMonotoneMinimalPerfectHashFunction.html
- Incremental Prefix hash for BitStrings (to have constant handle lookup in zft)
- We REALLY need images to illustrate
  - how rloc works
  - 6 children hack in AZFT
- check structs, check if we can make less random IO acceses, see /home/andrei/Thesis/mmph/go-boomphf-bs/boomphf-flat-arrays.go
- maybe compress uints in mph
- use instead rotl https://pkg.go.dev/math/bits#ReverseBytes16
  - Homework: check Assembly
    - go test -c
    objdump -d ./go-boomphf-bs.test
  - check repo for math.bits applications
- Update mph using one of suggested impls
  - see docs
    - https://docs.google.com/document/d/15lVjff73MiWJJczNcxw-YwEa4Gsk7uJfP3QBrcZyMfI/edit?tab=t.0
    - https://docs.google.com/document/d/1MUGEStgsORBztSGaef-edOMQJ0011Mo5fj8iWZKNHdg/edit?pli=1&tab=t.0
- Homework: double test changes on ARM & x86
- add 2 fattest links
- get rid of BitString interface

## Literature

#### That is the main article

- [Fast prefix search in little space, with applications](https://arxiv.org/abs/1804.04720)
    - Hollow ZFast Trie - data structure descriptions
- [Approximate Range Emptiness in Constant Time and Optimal Space](https://arxiv.org/pdf/1407.2907)
    - Proof of Range Emptiness lower bound
    - ApproximateRangeEmptiness structure
- [ZFastTrie & MonotoneMinimalPerfectHashing](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf)
    - see also [Learned Monotone Minimal Perfect Hashing](https://arxiv.org/pdf/2304.11012)

#### Related works

- [Learned Monotone Minimal Perfect Hashing](https://arxiv.org/pdf/2304.11012)
- [EXACT AND APPROXIMATE MEMBERSHIP TESTERS](http://aturing.umcs.maine.edu/~markov/Membership.pdf)
- [An Optimal Bloom Filter Replacement](https://arxiv.org/pdf/0804.1845)
- [Succincter](https://sci-hub.se/https://ieeexplore.ieee.org/abstract/document/4690964)
- [Succinct Range Filters](https://db.cs.cmu.edu/papers/2019/20_srf-zhang.pdf)
- [How to approximate a set](https://arxiv.org/abs/1304.1188)
- [Storing a Sparse Table with O(1) Worst Case Access Time](https://sci-hub.se/10.1145/828.1884)
- [SuRF: Practical Range Query Filtering with Fast Succinct Tries](https://db.cs.cmu.edu/papers/2018/mod601-zhangA-hm.pdf)
- [Rosetta: A Robust Space-Time Optimized Range Filter for Key-Value Stores](https://chatterjeesubarna.github.io/files/rosetta.pdf)
- [Memento Filter: A Fast, Dynamic, and Robust Range Filter](https://arxiv.org/abs/2408.05625)
- [SNARF: A Learning-Enhanced Range Filter](https://www.vldb.org/pvldb/vol15/p1632-vaidya.pdf)
- [Oasis: An Optimal Disjoint Segmented Learned Range Filter](https://www.vldb.org/pvldb/vol17/p1911-luo.pdf)
- [Proteus: A Self-Designing Range Filter](https://arxiv.org/abs/2207.01503)
