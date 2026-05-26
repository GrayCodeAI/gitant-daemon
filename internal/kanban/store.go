package kanban

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Board represents a Kanban board
type Board struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Columns     []Column  `json:"columns"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Column represents a Kanban column
type Column struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Limit   int    `json:"limit"` // WIP limit
	Order   int    `json:"order"`
	Cards   []Card `json:"cards"`
}

// Card represents a Kanban card
type Card struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee"`
	Labels      []string  `json:"labels"`
	IssueID     string    `json:"issue_id,omitempty"`
	PRID        string    `json:"pr_id,omitempty"`
	Priority    string    `json:"priority"` // "low", "medium", "high", "critical"
	DueDate     *time.Time `json:"due_date,omitempty"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store manages Kanban boards
type Store struct {
	mu      sync.RWMutex
	baseDir string
	boards  map[string][]*Board
}

// NewStore creates a new Kanban store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		boards:  make(map[string][]*Board),
	}
}

// Load loads boards from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "kanban.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.boards)
}

// Save saves boards to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "kanban.json")
	data, err := json.MarshalIndent(s.boards, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new board
func (s *Store) Create(board *Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	board.ID = fmt.Sprintf("board-%d", time.Now().UnixNano())
	board.CreatedAt = time.Now()
	board.UpdatedAt = time.Now()

	// Default columns
	if len(board.Columns) == 0 {
		board.Columns = []Column{
			{ID: "backlog", Name: "Backlog", Order: 0, Cards: []Card{}},
			{ID: "todo", Name: "To Do", Order: 1, Cards: []Card{}},
			{ID: "in-progress", Name: "In Progress", Order: 2, Cards: []Card{}, Limit: 5},
			{ID: "review", Name: "Review", Order: 3, Cards: []Card{}},
			{ID: "done", Name: "Done", Order: 4, Cards: []Card{}},
		}
	}

	s.boards[board.RepoID] = append(s.boards[board.RepoID], board)
	return s.Save()
}

// Get gets a board by ID
func (s *Store) Get(repoID, boardID string) (*Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.boards[repoID] {
		if b.ID == boardID {
			return b, nil
		}
	}
	return nil, fmt.Errorf("board not found")
}

// List lists boards for a repository
func (s *Store) List(repoID string) []*Board {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.boards[repoID]
}

// AddCard adds a card to a column
func (s *Store) AddCard(repoID, boardID, columnID string, card *Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.boards[repoID] {
		if b.ID == boardID {
			for i, col := range b.Columns {
				if col.ID == columnID {
					// Check WIP limit
					if col.Limit > 0 && len(col.Cards) >= col.Limit {
						return fmt.Errorf("WIP limit reached for column %s", columnID)
					}

					card.ID = fmt.Sprintf("card-%d", time.Now().UnixNano())
					card.CreatedAt = time.Now()
					card.UpdatedAt = time.Now()
					card.Order = len(col.Cards)
					if card.Labels == nil {
						card.Labels = []string{}
					}

					b.Columns[i].Cards = append(b.Columns[i].Cards, *card)
					b.UpdatedAt = time.Now()
					return s.Save()
				}
			}
			return fmt.Errorf("column not found")
		}
	}
	return fmt.Errorf("board not found")
}

// MoveCard moves a card between columns
func (s *Store) MoveCard(repoID, boardID, cardID, targetColumnID string, targetOrder int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.boards[repoID] {
		if b.ID == boardID {
			// Find and remove card
			var card *Card
			for i, col := range b.Columns {
				for j, c := range col.Cards {
					if c.ID == cardID {
						card = &c
						b.Columns[i].Cards = append(col.Cards[:j], col.Cards[j+1:]...)
						break
					}
				}
				if card != nil {
					break
				}
			}

			if card == nil {
				return fmt.Errorf("card not found")
			}

			// Add to target column
			for i, col := range b.Columns {
				if col.ID == targetColumnID {
					// Check WIP limit
					if col.Limit > 0 && len(col.Cards) >= col.Limit {
						return fmt.Errorf("WIP limit reached for column %s", targetColumnID)
					}

					card.Order = targetOrder
					card.UpdatedAt = time.Now()
					b.Columns[i].Cards = append(b.Columns[i].Cards, *card)
					b.UpdatedAt = time.Now()
					return s.Save()
				}
			}

			return fmt.Errorf("target column not found")
		}
	}
	return fmt.Errorf("board not found")
}

// UpdateCard updates a card
func (s *Store) UpdateCard(repoID, boardID, cardID string, fn func(*Card) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.boards[repoID] {
		if b.ID == boardID {
			for i, col := range b.Columns {
				for j, card := range col.Cards {
					if card.ID == cardID {
						if err := fn(&b.Columns[i].Cards[j]); err != nil {
							return err
						}
						b.Columns[i].Cards[j].UpdatedAt = time.Now()
						b.UpdatedAt = time.Now()
						return s.Save()
					}
				}
			}
			return fmt.Errorf("card not found")
		}
	}
	return fmt.Errorf("board not found")
}

// DeleteCard deletes a card
func (s *Store) DeleteCard(repoID, boardID, cardID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.boards[repoID] {
		if b.ID == boardID {
			for i, col := range b.Columns {
				for j, card := range col.Cards {
					if card.ID == cardID {
						b.Columns[i].Cards = append(col.Cards[:j], col.Cards[j+1:]...)
						b.UpdatedAt = time.Now()
						return s.Save()
					}
				}
			}
			return fmt.Errorf("card not found")
		}
	}
	return fmt.Errorf("board not found")
}
