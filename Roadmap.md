## Plan

- [ ] `SuccinctBitVector` Лаконичные Битовые Векторы
- [ ] `LocalExactRangeStructure`
    - [ ] Weak Prefix Search `Hollow Z Fast Trie`
        - [x] Deterministic Z Fast Trie
        - [ ] Probabilistic Z Fast Trie
        - [ ] MMPH (Монтонное Минимальное Совершенное Хеширование)
        - [ ] Encoded Handles table T (handles -> len(e_alpha))
            - [ ] ...
    - [ ] Range Locator
- [ ] `ExactRangeEmptiness`
- [ ] `ApproximateRangeEmptiness`

### `SuccinctBitVector` Лаконичные Битовые Векторы [succinct_bit_vector.md](succinct_bit_vector/SuccinctBitVector.md)

Succinct Bit Vector — это пространственно-эффективная структура данных, позволяющая хранить битовый массив длины $N$, занимая $N + o(N)$
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

### Z Fast Trie [z_fast_trie.md](zfasttrie/ZFastTrie.md)

#### Api

`interface ZFastTrie`

- `Build(keys []string)`
- `PrefixSearch(query string) PrefixSearchResult` - for example range [L, R] in Weak Prefix Search Task
- `PredecessorSearch(query string) PredecessorSearchResult` - for example to find bucket in MMPH

#### Deterministic Implementation [zfasttrie](zfasttrie) was adopted from [ctriepp](https://gitlab.com/habatakitai/ctriepp)

### MMPH (Monotone Minimal Perfect Hashing) [mmph/MonotoneMinimalPerfectHashing.md](mmph/MonotoneMinimalPerfectHashing.md)

Биективное отображение элементов отсортированного множества ключей в их порядковые номера с сохранением лексикографического порядка.

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

## Literature

#### That is the main article

- [Approximate Range Emptiness in Constant Time and Optimal Space](https://arxiv.org/pdf/1407.2907)

#### Related works

- [EXACT AND APPROXIMATE MEMBERSHIP TESTERS](http://aturing.umcs.maine.edu/~markov/Membership.pdf)
- [An Optimal Bloom Filter Replacement](https://arxiv.org/pdf/0804.1845)
- [Succincter](https://sci-hub.se/https://ieeexplore.ieee.org/abstract/document/4690964)
- [Fast prefix search in little space, with applications](https://arxiv.org/abs/1804.04720)
- [ZFastTrie & MonotoneMinimalPerfectHashing](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf)
- [Succinct Range Filters](https://db.cs.cmu.edu/papers/2019/20_srf-zhang.pdf)
- [How to approximate a set](https://arxiv.org/abs/1304.1188)
- [Storing a Sparse Table with O(1) Worst Case Access Time](https://sci-hub.se/10.1145/828.1884)