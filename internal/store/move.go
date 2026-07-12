package store

import "fmt"

// CheckMoveCycle rejects a new parent that is the task itself or any of its
// descendants — BuildTree would silently drop the whole cycle from view.
func (s *Store) CheckMoveCycle(taskID, newParentID int64) error {
	seen := map[int64]bool{}
	for id := newParentID; id != 0; {
		if id == taskID {
			return fmt.Errorf("movimento criaria um ciclo: %d é descendente de %d", newParentID, taskID)
		}
		if seen[id] {
			return fmt.Errorf("hierarquia corrompida: ciclo existente em %d", id)
		}
		seen[id] = true
		p, err := s.GetTask(id)
		if err != nil {
			return fmt.Errorf("pai %d não existe", id)
		}
		id = p.ParentID
	}
	return nil
}
