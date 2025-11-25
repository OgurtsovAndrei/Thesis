# Monotone Minimal Perfect Hashing (MMPH)

## Обзор

**Monotone Minimal Perfect Hashing (MMPH)** — это статическая сжатая (succinct) структура данных, которая обеспечивает биективное отображение элементов отсортированного множества ключей $S$ в их порядковые номера (ранги) в диапазоне $[0, |S|-1]$.

В отличие от обычного хеширования, MMPH сохраняет лексикографический порядок ключей:

$$\forall x, y \in S: x < y \implies MMPH(x) < MMPH(y)$$

## Два варианта реализации

### Вариант A: Time-optimized MMPH (Section 3)
- **Query Time**: $O(1)$ (детерминированное)
- **Space**: $O(n \log w)$ бит  
- **Техника**: Bucketing with Longest Common Prefixes (LCP)
- **Применение**: Когда критично время запроса (например, Range Locator)

### Вариант B: Space-optimized MMPH (Section 6)  
- **Query Time**: $O(\log w)$ 
- **Space**: $O(n \log \log w)$ бит
- **Техника**: Bucketing by relative ranking + probabilistic trie
- **Применение**: Когда критична память

**Общие характеристики**:
- **Тип**: Статическая структура (строится один раз на неизменяемом наборе данных)
- **Время построения**: $O(n \log w)$ для обоих вариантов

## API

Структура предоставляет следующий минималистичный интерфейс:

### `Build(sorted_keys)`

Строит индекс на основе переданного набора уникальных отсортированных ключей.

- **Вход**: Массив ключей (статический отсортированный список)
- **Сложность**: $O(n \log w)$

### `Rank(key) -> int`

Возвращает порядковый номер (ранг) ключа в исходном множестве.

- **Вход**: Ключ $x$
- **Выход**: Целое число $i \in [0, n-1]$
- **Сложность**: $O(1)$ для варианта A, $O(\log w)$ для варианта B

**⚠️ Важно**: Если $key \notin S$, результат не определен (возвращается произвольное значение). Валидация принадлежности должна выполняться внешними компонентами (например, Bloom Filter).

## Область применения

В архитектуре Range Filter (задача Approximate Range Emptiness) MMPH используется как низкоуровневый строительный блок для ускорения навигации.

### Иерархия использования

```
Approximate Range Emptiness (Верхний уровень)
            ↓
Local Exact Range Structure (Хранилище сжатых ключей)
            ↓
Weak Prefix Search (Поиск узла в неявном префиксном дереве)
            ↓
Range Locator (Преобразование узла дерева в диапазон индексов массива)
            ↓
MMPH (Непосредственное вычисление индексов границ диапазона за O(1))
```

MMPH позволяет заменить бинарный поиск ($O(\log n)$) на константное вычисление адреса данных, что критично для общей производительности фильтра.

## Ссылки и Литература

### Основной алгоритм (Bucketing with LCP)
- [Monotone Minimal Perfect Hashing: Searching a Sorted Table with O(1) Accesses](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf) - Belazzougui D., Boldi P., Pagh R., Vigna S. (Раздел 3)

### Контекст использования (Weak Prefix Search)
- [Fast Prefix Search in Little Space, with Applications](https://arxiv.org/abs/1804.04720) - Belazzougui D., Boldi P., Pagh R., Vigna S.

### Родительская задача (Approximate Range Emptiness)
- [Approximate Range Emptiness in Constant Time and Optimal Space](https://arxiv.org/pdf/1407.2907) - Goswami M., Grønlund A., Larsen K. G., Pagh R.