package session

import (
	"container/heap"
	"fmt"
	"strings"
)

type QueueItem struct {
	ID         uint
	Name       string
	Command    string
	Dependency uint64
	Check      func(*Session) bool
	Priority   int
	index      int // needed for container/heap interface
}

type PriorityQueue []*QueueItem

type QueueRegistry struct {
	Queue     PriorityQueue
	LastIndex uint
}

func NewQueueRegistry() *QueueRegistry {
	qr := &QueueRegistry{Queue: make(PriorityQueue, 0), LastIndex: 0}
	heap.Init(&qr.Queue)
	return qr
}

// add a new item to the queue and return the ID of the item
// This is convenient to add task chains
func (q *QueueRegistry) Add(item *QueueItem) uint {
	item.ID = q.LastIndex
	q.LastIndex++
	heap.Push(&q.Queue, item)
	return item.ID
}

func (q *QueueRegistry) Len() int {
	return q.Queue.Len()
}

// Get sorted queue without emptying it
func (q *QueueRegistry) ViewQueue() []*QueueItem {
	tempQueue := make(PriorityQueue, 0)
	var queue []*QueueItem

	for q.Queue.Len() > 0 {
		item := heap.Pop(&q.Queue).(*QueueItem)
		queue = append(queue, item)
		tempQueue = append(tempQueue, item)
	}

	for tempQueue.Len() > 0 {
		heap.Push(&q.Queue, heap.Pop(&tempQueue))
	}

	return queue
}

// This function will return the first item that is ready to be processed
func (s *Session) GetQueueItem() *QueueItem {
	tempQueue := make(PriorityQueue, 0)

	for s.Queue.Queue.Len() > 0 {
		item := heap.Pop(&s.Queue.Queue).(*QueueItem)
		if item.Check == nil || item.Check(s) {
			// Push the items back to the main queue
			for tempQueue.Len() > 0 {
				heap.Push(&s.Queue.Queue, heap.Pop(&tempQueue))
			}
			return item
		} else {
			heap.Push(&tempQueue, item)
		}
	}

	// Push the items back to the main queue if no item was found
	for tempQueue.Len() > 0 {
		heap.Push(&s.Queue.Queue, heap.Pop(&tempQueue))
	}

	return nil
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueItem)
	item.index = n
	*pq = append(*pq, item)
	//heap.Fix(pq, n)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func CmdQueue(s *Session, cmd string) {
	var builder strings.Builder

	builder.WriteString("ID\tName\tCommand\tDependency\tPriority\n")
	for _, item := range s.Queue.ViewQueue() {
		line := fmt.Sprintf("%d\t%s\t%s\t%d\t%d\n", item.ID, item.Name, item.Command, item.Dependency, item.Priority)
		builder.WriteString(line)
	}

	for s.Queue.Queue.Len() > 0 {
		item := s.GetQueueItem()
		line := fmt.Sprintf("Run %s\n", item.Name)
		builder.WriteString(line)
	}

	s.Output(builder.String())
}
