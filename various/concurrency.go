package various

import "sync"

func KickOffChunkWorkers(totalItems int, fn func(start, end int)) {
	numWorkers := 8

	var wg sync.WaitGroup
	var chunkStart int
	chunkSize := (totalItems / numWorkers) + 1
	for i := 0; i < numWorkers; i++ {
		curChunk := chunkSize
		if rem := totalItems - chunkStart; rem < curChunk {
			curChunk = rem
		}
		if curChunk <= 0 {
			break
		}
		wg.Add(1)
		go func(start, end int) {
			fn(start, end)
			wg.Done()
		}(chunkStart, chunkStart+curChunk)
		chunkStart += curChunk
	}
	wg.Wait()
}
