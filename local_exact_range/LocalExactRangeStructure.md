# План реализации LocalExactRangeStructure

**LocalExactRangeStructure** — это вспомогательная структура данных, описанная в разделе 3.2 статьи [*"Approximate Range Emptiness in Constant Time and Optimal Space"*](https://arxiv.org/pdf/1407.2907). Она представляет собой сжатую (succinct) структуру для точных запросов на пустоту диапазона (Range Emptiness). В контексте общей задачи она используется для хранения точек после их отображения (хеширования) в уменьшенный универсум.

## 1. API и Асимптотика

Эта структура хранит множество точек $S$ из универсума $[U]$ и отвечает на запросы существования точек в диапазоне $[a, b]$.

### Интерфейс (API)

#### `Build(points []int, U int)`

Строит структуру на основе отсортированного списка уникальных точек.

- **Вход**: массив точек `points`, размер универсума `U` (в контексте ApproximateRE это будет $r = nL/\epsilon$)

#### `IsEmpty(a int, b int) -> bool`

- Возвращает `true`, если в диапазоне $[a, b]$ нет точек из $S$
- Возвращает `false`, если пересечение $[a, b] \cap S \neq \emptyset$

#### `Report(a int, b int) -> []int` (опционально, упоминается в статье)

Возвращает список всех точек в диапазоне.

### Асимптотика

#### Space (Память)
$n \lg(U/n) + O(n \lg^\delta(U/n))$ бит.

Это близко к теоретическому минимуму. Младший член $O(n \lg^\delta(U/n))$ возникает из-за использования структуры Weak Prefix Search с константным временем запроса.

#### Query Time (Время запроса)
$O(1)$.

Константное время достигается за счет разбиения универсума на поддиапазоны и использования Weak Prefix Search для определения границ.

## 2. Пререквизиты (Вспомогательные структуры)

Для реализации LocalExactRangeStructure потребуется реализовать несколько низкоуровневых сжатых примитивов и структур, описанных в связанных статьях (в частности, в *"Fast Prefix Search in Little Space"*).

### A. Rank & Select (Сжатые битовые массивы)

Необходимы для навигации по разбитому универсуму.

- **BitVector**: Должен поддерживать операции `Rank1(i)` (количество единиц до индекса i) и `Select1(i)` (индекс i-й единицы) за $O(1)$
- Используется в структурах $D_1$ и $D_2$ для управления поддиапазонами $s_i$
- Детали реализации в [SuccinctBitVector.md](../succinct_bit_vector/SuccinctBitVector.md)

### B. Weak Prefix Search (Слабый поиск префикса)

Это ключевой компонент. Ссылка [3] в статье указывает на [*"Fast Prefix Search in Little Space"*](https://arxiv.org/abs/1804.04720).

**Задача**: Для префикса $p$ вернуть диапазон рангов $[i, j)$ строк (точек), имеющих этот префикс.

**Реализация**:
- **Compact Trie** (сжатое префиксное дерево): Логическое представление ключей
- **Hollow Z-Fast Trie**: Структура для отображения префикса в узел дерева ("exit node"). Использует хеширование. Детали в [ZFastTrie.md](../zfasttrie/z_fast_trie.md)
- **Range Locator**: Определяет диапазон листьев под найденным узлом. Требует Monotone Minimal Perfect Hashing (MMPH)

### C. Monotone Minimal Perfect Hashing (MMPH)

- Необходим внутри Range Locator
- Позволяет отображать ключи в их ранги с сохранением порядка
- Ссылка на [*"Monotone Minimal Perfect Hashing"*](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf) (Belazzougui et al.)

## 3. Место в ApproximateRangeEmptiness

LocalExactRangeStructure является "бэкендом" для вероятностной структуры.

### Universe Reduction (Сжатие универсума)

- Входные точки из большого универсума $[U]$ отображаются в меньший универсум $[r]$, где $r = nL/\epsilon$
- Используется хеш-функция, сохраняющая локальность: $h(x) = (u(\lfloor x/r \rfloor) + x) \mod r$

### Хранение

- Хешированные точки $h(S)$ сохраняются именно в LocalExactRangeStructure
- Поскольку размер нового универсума $r$ линейно зависит от $n$, структура потребляет $O(n \lg(L/\epsilon))$ бит

### Выполнение запроса

- Запрос $[a, b]$ в оригинальной структуре трансформируется в (максимум два) интервала в уменьшенном универсуме $[r]$
- `LocalExactRangeStructure.IsEmpty` проверяет эти интервалы
- Если они пусты, возвращается "Empty"
- Если нет — "Non-Empty" (с возможной ошибкой false positive из-за коллизий хеша)

## Итоговый чек-лист для реализации

- [ ] Реализовать Rank/Select ([SuccinctBitVector.md](../succinct_bit_vector/SuccinctBitVector.md))
- [ ] Реализовать MMPH ([Monotone Minimal Perfect Hashing](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf))
- [ ] Реализовать Weak Prefix Search поверх MMPH ([Fast Prefix Search](https://arxiv.org/abs/1804.04720))
- [ ] Собрать LocalExactRangeStructure:
  - [ ] Разбить универсум на $n$ частей
  - [ ] Построить битовые карты $D_1, D_2$
  - [ ] Для каждого непустого поддиапазона построить инстанс Weak Prefix Search
  - [ ] Реализовать логику `IsEmpty` через поиск префиксов (LCP запроса)

## Литература

### Основная статья
- [Approximate Range Emptiness in Constant Time and Optimal Space](https://arxiv.org/pdf/1407.2907)

### Связанные работы
- [Fast prefix search in little space, with applications](https://arxiv.org/abs/1804.04720)
- [Monotone Minimal Perfect Hashing](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf)