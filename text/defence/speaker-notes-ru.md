# Заметки докладчика — Защита диссертации

---

## Слайд 0 — Титульный + арка выступления (0:30)

**Пример текста (~25с):**
> "Тема моей диссертации — «range emptiness фильтры».

---

## Слайд 1 — LSM-дерево (Log-Structured Merge-Tree) (0:50)

**Пример текста (~50с):**
> «Тема работы — range emptiness фильтры которые используются в LSM-деревьях.
>
> LSM — write-optimized струтура данных, используемая в RocksDB, LevelDB, Cassandra.
> In LSM-tree keys are stored in different levels in immutable sorted files — SSTable's.
> Размер файлов на уровнях растёт геометрически. Inserts идут в memtable in RAM; когда она
> заполняется — сбрасывается на диск как новый SSTable на нулевом уровне. Когда на уровне накапливается слишком много
> файлов — они компактируются in background: сливаются вместе и переходят на следующий уровень.
> Из-за sequential I/O, compaction быстрый.
>
> При чтении данных картина другая: lookup вынужден обратиться к каждому SSTable в дереве.»

---

## Слайд 2 — Снижение дискового I/O с помощью фильтров (1:15)

**Пример текста (~45с):**
> Чтобы не читать каждый SSTable с диска при lookup-е, в метаданных каждого файла живёт фильтр.
> Он отвечает с односторонней ошибкой не больше ε: false negative-ов нет, false positive-ы допустимы.
> Этого достаточно — если фильтр говорит "нет", база данных
> пропускает SSTable без disk I/O.
> One of widly used - Bloom filter решает point queries.
>
> *— листаем слайд —*

---

## Слайд 3— Проблема и базовое решение Госвами (1:30)

**Ключевые моменты:**

**Пример текста (~75с):**
> «Range filter is the structure for range queries. Instead of answering «does the set contains the key» — он отвечает:
> «if a range [a, b] intercept the set?». Те же гарантии: нет false negative-ов, false positive rate не выше ε.
> Именно такой фильтр я строю.
>
> Формально: дано множество S из n ключей, максимальная длина запроса 𝓛, целевой false positive rate ε.
>
> Госвами доказал информационно-теоретическую нижнюю границу для любого S — log₂(𝓛/ε) бит на ключ, и привел
> конструкцию, которая достигает этой нижней границы с точностью до константы.
> And constructed a DS which reaches this lower limit up to a constant.
> As for the computational complexity, this data structure answers intersection queries in constant time.
> Их конструкция:
> - locality-preserving hash that compresses the universe, and
> - exact range emptiness filter (ERE filter) which answers intersection queries exactly.
    > Асимптотически оптимально — но именно константы за этой асимптотой я атакую в своей работе.»

---

## Слайд 5 — Exact Range Emptiness as Elias-Fano (1:00)

**Ключевые моменты:**

**Пример текста (~55с):**
> «Разберём ERE на примере. To keep things simple, we will only consider an example on this slide.
> Here we have 8 - 8 bit keys.
> Делим каждый ключ на 3-битный префикс (log 8) и 5-битный суффикс — префикс определяет bucket, суффикс в нём хранится.
> Все суффиксы лежат в packed массиве A, размер которого — K минус log n бит на ключ, где K - размер ключа.
>
> Для навигации нужно знать, где начинается каждый бакет. Госвами хранил два вектора:
> - D₁ — какие бакеты непустые,
> - D₂ — размеры непустых бакетов в unary.
    > Навигация через D₁ и D₂ вместе.
    > I replaced [Goswami's two vectors] with one succinct vector which encodes bucket
    lengths [in unary encoding, including empty buckets].
    > Result vector size is 2n+1 бит.
    > This is Elias-Fano encoding for sorted set of keys.
>
> To answer a query [a, b]:
> The intersection of [a, b] with S reduces to checking at most two boundary buckets.
> We navigate to them using Select(D, i) in O(1), then run adaptive binary/linear search
> on packed suffixes inside each bucket.
> Goswami uses Weak Prefix Search instead.

---

## Слайд 6 — Компактный бэкенд (1:00) — центральная часть

**Пример частей текста на английском / улучшения:**
> "WPS is O(1), but this O(1) depends on two important factors when translated to nanoseconds on real hardware:
> (1) working set size: — whether the WPS auxiliary structure fits in the CPU cache;
> (2) memory access pattern of the algorithm: — sequential or random.
> Structures with a small working set that fits in cache are faster. But more importantly,
> sequential memory access patterns outperform random access on real hardware regardless of dataset size."
>
> "The first row [Facebook median, k=27] shows a case where the WPS structure fits in L1 cache;
> the other three rows don't fit — and the gap is large.
> The same large performance gap arises when we replace random memory accesses with sequential reads.
> That is why replacing Goswami's two-level metadata with a single contiguous vector is a very important
> hardware optimization — sequential accesses improve performance on structures of any size."
>
> "At practical dataset sizes — 2²⁴ to 2²⁸ keys — buckets are small enough that Binary or even linear search,
> with their sequential memory access patterns, outperforms WPS, and do not require auxiliary index."

---

## Слайд 7 — Реальные распределения кластеризованы (0:40) — связка

**Пример текста (~40с):**

> "The performance of approximate range filters depends on two aspects:
> the performance of the ERE backend at the bottom level, and the choice of locality-preserving hash
> that can reduce the approximate emptiness problem to multiple exact emptiness problems."
>
> On the previous slide I did consider how to speed up Exact Range Empt Filter
>
> Now I will discuss how we can choose locality-preserving hashes in accordance with
> the distribution of keys in our dataset.
>
> Goswami bound works for any distribution of keys.
> Btw, in many practical workloads, keys tend to be clustered. [На гистограмму]
> Dense clusters alternate with sparse regions.
> For examples: Файловые пути, urls, S2 Cell IDs.
>
> So, let's divide the universe into parts — dense clusters and sparse gaps — will be handled differently.»

---

## Слайд 8 — Два режима: плотный кластер и разреженный хвост (~0:55)

**Пример частей текста на английском / улучшения:**

> There are two different scenarios,
>
> We can have a dense cluster with many keys in a small window.  
> Then, if the compressed universe in Goswami's hash is larger than the original cluster window,
> we don't need a hash at all — we can build an exact range emptiness filter directly on the window.
> It costs fewer bits per key and gives zero false positive rate.
>
> "In sparse regions where we have very few points, we can simply truncate some lower bits
> to obtain our locality-preserving hash. And this hash will still have few collisions."

---

## Слайд 9 — Обнаружение: 1D-DBSCAN по отсортированным ключам (~0:40)

**Пример текста (~40с):**
> Остался один вопрос: как найти разбиение.
> Для решения этой задачи мы применяем алгоритм кластеризации DBSCAN.
> На отсортированных 1D ключах DBSCAN работает за линейное время.
>
> Каждый плотный сегмент переходит в ERE.
> Sparse точки которые DBSCAN определяет как шум - идут в truncation.

---

## Слайд 10 — Главный результат: SOSD Facebook, 𝓛=128, n=2²⁴ (1:15)

**Пример текста (~55с):**
> "Главный результат.
> На графике — зависимость FPR от BPK.
> SNARF, SuRF и Rosetta — другие решения той же задачи approximate range emptiness.
> Структуры построены на Facebook user IDs из датасета SOSD (Search on Sorted Data).
> n = 2²⁴ — тот же масштаб, на котором SuRF показывает свои лучшие результаты.
> DB-scan Range Filter (пурпурная линия) достигает нулевого FPR на 11 битах на ключ.
> SNARF насыщается на уровне 7×10⁻⁴ — сколько бы памяти ему ни дали, ниже не опускается."

## Слайд 11 — Ограничения: Равномерное распределение, 𝓛=1, n=2²⁴ (0:30)

**Пример текста (~30с):**
> "Честное ограничение: где мы не выигрываем.
> Равномерные ключи, точечные запросы.
> Нет кластеров для сегментации - нет диапазонов для выгоды от отсечения.
> Мы же платим за накладные расходы сегментации, ничего не сегментируя.
> Наше преимущество — на реальных данных."

---

## Слайд 12 — Заключение: Три результата (1:00)

**Пример текста (~50с):**
> "Три главных итога.
> Мы сократили использование метаданных на 24% — за счёт одновекторного Elias-Fano ERE.
> Мы сократили query latency в 13–30 раз — за счёт замены WPS на адаптивный linear / бинарный поиск.
> Мы достигли FPR 0 на 11 битах на ключ на Facebook user IDs за счёт DB-scan Range Filter.
> Будущая работа: динамические обновления, Partial Elias-Fano encoding, интеграция в RocksDB. Спасибо за внимание."
