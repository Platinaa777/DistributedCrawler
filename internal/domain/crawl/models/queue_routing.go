package models

import "math/rand"

// QueueWeight specifies a routing weight for a named crawl queue.
// Weight 0 excludes the queue from routing. Queues not listed get weight 1.
type QueueWeight struct {
	Queue  string `json:"queue"`
	Weight uint32 `json:"weight"`
}

// SelectCrawlQueue returns one queue name using weighted random selection.
// When weights is non-empty, routes strictly within those queues (queues with Weight==0 are excluded).
// When weights is empty, selects uniformly from available.
// Returns "" when there are no candidates.
func SelectCrawlQueue(available []string, weights []QueueWeight) string {
	type candidate struct {
		queue  string
		weight uint32
	}

	// Per-job weights act as the full routing spec when provided.
	if len(weights) > 0 {
		var candidates []candidate
		totalWeight := uint32(0)
		for _, w := range weights {
			if w.Weight == 0 {
				continue // explicitly excluded
			}
			candidates = append(candidates, candidate{queue: w.Queue, weight: w.Weight})
			totalWeight += w.Weight
		}
		if len(candidates) == 1 {
			return candidates[0].queue
		}
		if len(candidates) > 1 {
			r := uint32(rand.Intn(int(totalWeight)))
			for _, c := range candidates {
				if r < c.weight {
					return c.queue
				}
				r -= c.weight
			}
			return candidates[len(candidates)-1].queue
		}
		// All weights were 0 — fall through to available.
	}

	// No usable weights: uniform distribution over available queues.
	if len(available) == 0 {
		return ""
	}
	if len(available) == 1 {
		return available[0]
	}
	return available[rand.Intn(len(available))]
}
