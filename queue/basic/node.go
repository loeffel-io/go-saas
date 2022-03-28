package makeless_go_queue_basic

import (
	"github.com/makeless/makeless-go/v2/queue"
	"sync"
)

type Node struct {
	Data []byte

	next makeless_go_queue.Node
	*sync.RWMutex
}

func (node *Node) GetData() []byte {
	node.RLock()
	defer node.RUnlock()

	return node.Data
}

func (node *Node) GetNext() makeless_go_queue.Node {
	node.RLock()
	defer node.RUnlock()

	return node.next
}

func (node *Node) SetNext(next makeless_go_queue.Node) {
	node.Lock()
	defer node.Unlock()

	node.next = next
}
