package domain

import "sort"

// Reach описывает, куда может дойти ОДНА шашка игрока s.Turn, если двигать
// только её, тратя кубики из s.Dice.Remaining по порядку.
//
// From — исходный пункт шашки (1..24).
// Path — последовательность пунктов-остановок по порядку; каждый элемент —
//
//	цель отдельного MOVE. Длина >= 1, последний элемент — финальная цель.
//
// Pips — потраченные пипсы по порядку, len(Pips) == len(Path).
//
// Для одиночного хода Path и Pips длины 1 (как обычная подсказка). Для
// составного хода (несколько кубиков одной шашкой) — длины 2..4.
//
// Выкид (To == 0) в Reach НЕ попадает: это отдельная зона UI, и шашка после
// выкида исчезает (цепочку продолжать нечем).
type Reach struct {
	From Point
	Path []Point
	Pips []uint8
}

// ReachableTargets возвращает все достижимые цели каждой шашки игрока s.Turn,
// включая составные ходы одной шашкой несколькими кубиками. Для пары
// (From, финальная To) оставляется одна (кратчайшая найденная) цепочка.
//
// Шаг цепочки включается только если ПОСЛЕ него существует легальное
// завершение хода (canReachLegalFinal) — та же гарантия, что в LegalMoves:
// мы не предлагаем встать в позицию, из которой END_TURN невозможен. Базовые
// правила, направление и правило головы учитываются через candidateMoves;
// правило шести на финале — через canReachLegalFinal.
//
// Результат отсортирован детерминированно: по From, затем по числу кубиков,
// затем по финальной точке — чтобы вывод был стабилен (важно для тестов и для
// читаемости протокола).
func ReachableTargets(s GameState) []Reach {
	var out []Reach
	seen := map[[2]Point]bool{}
	for from := Point(1); from <= 24; from++ {
		if !hasCheckerOf(s.Board, s.Turn, from) {
			continue
		}
		reachFrom(s, from, from, nil, nil, seen, &out)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if len(out[i].Pips) != len(out[j].Pips) {
			return len(out[i].Pips) < len(out[j].Pips)
		}
		return out[i].Path[len(out[i].Path)-1] < out[j].Path[len(out[j].Path)-1]
	})
	return out
}

// reachFrom — DFS по ходам одной шашки. cur — где шашка сейчас, path/pips —
// накопленная цепочка от origin. Перебирает только ходы из cur (одна и та же
// шашка), исключая выкид; для каждого валидного шага, не заводящего в тупик
// без легального финала, записывает цель и углубляется.
func reachFrom(s GameState, origin, cur Point, path []Point, pips []uint8, seen map[[2]Point]bool, out *[]Reach) {
	for _, m := range candidateMoves(s) {
		if m.From != cur || m.To == 0 {
			continue
		}
		ns, err := Apply(s, m)
		if err != nil {
			continue
		}
		if !canReachLegalFinal(ns, s.Turn) {
			continue
		}
		nextPath := append(append([]Point{}, path...), m.To)
		nextPips := append(append([]uint8{}, pips...), m.Pip)
		key := [2]Point{origin, m.To}
		if !seen[key] {
			seen[key] = true
			*out = append(*out, Reach{From: origin, Path: nextPath, Pips: nextPips})
		}
		reachFrom(ns, origin, m.To, nextPath, nextPips, seen, out)
	}
}
