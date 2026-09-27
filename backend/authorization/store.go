package authorization

import (
	"slices"
	"sync"
)

// Store keeps processed transactions in memory so their attempt chains can be
// queried for analytics. It is safe for concurrent use.
type Store struct {
	mu           sync.RWMutex
	transactions []Transaction
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Save(txn Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactions = append(s.transactions, txn)
}

// List returns every stored transaction, oldest first.
func (s *Store) List() []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.transactions)
}
