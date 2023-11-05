package geo

// QueueEntry is a single entry in the priority queue.
type QueueEntry struct {
	Index       int     // index of the item in the heap.
	Score       float64 // priority of the item in the queue.
	Origin      int     // origin region / ID
	Destination int     // destination region / ID
}

// AscPriorityQueue implements heap.Interface and holds Items.
// Priority is ascending (lowest score first).
type AscPriorityQueue []*QueueEntry

func (pq AscPriorityQueue) Len() int { return len(pq) }

func (pq AscPriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	// return pq[i].score > pq[j].score // 3, 2, 1

	// We want Pop to give us the lowest, not highest, priority so we use less than here.
	return pq[i].Score < pq[j].Score // 1, 2, 3
}

func (pq *AscPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *AscPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueEntry)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq AscPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index, pq[j].Index = i, j
}
