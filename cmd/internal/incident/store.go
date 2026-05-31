package incident

import (
	"sync"
	"time"
)

type Stage string

const (
	StageDetecting   Stage = "detecting"
	StageAggregating Stage = "aggregating"
	StageDiagnosing  Stage = "diagnosing"
	StageSandboxing  Stage = "sandboxing"
	StagePending     Stage = "pending_approval"
	StageHealing     Stage = "healing"
	StageHealed      Stage = "healed"
	StageFailed      Stage = "failed"
)

type Incident struct {
	ID               string     `json:"id"`
	Repo             string     `json:"repo"`
	HeadSHA          string     `json:"head_sha"`
	Branch           string     `json:"branch"`
	Stage            Stage      `json:"stage"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	FailedJobs       string     `json:"failed_jobs"`
	CommitContext    string     `json:"commit_context"`
	DiagnosisSummary string     `json:"diagnosis_summary"`
	GeneratedDiff    string     `json:"generated_diff"`
	FilesModified    []string   `json:"files_modified"`
	ConfidenceScore  int        `json:"confidence_score"`
	PRUrl            string     `json:"pr_url"`
	HealedAt         *time.Time `json:"healed_at"`
	Error            string     `json:"error"`
	CoralSQLQuery    string     `json:"coral_sql_query"`
	BlastRadius      string     `json:"blast_radius"`
	approveCh        chan bool  `json:"-"` // This tag stops JSON from trying to encode the channel
}

type Store struct {
	mu        sync.RWMutex
	incidents map[string]*Incident
}

var Global = &Store{
	incidents: make(map[string]*Incident),
}

func (s *Store) Create(id, repo, sha, branch string) *Incident {
	inc := &Incident{
		ID:        id,
		Repo:      repo,
		HeadSHA:   sha,
		Branch:    branch,
		Stage:     StageDetecting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		approveCh: make(chan bool, 1),
	}
	s.mu.Lock()
	s.incidents[id] = inc
	s.mu.Unlock()
	return inc
}

func (s *Store) Get(id string) (*Incident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inc, ok := s.incidents[id]
	return inc, ok
}

func (s *Store) List() []*Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Incident, 0, len(s.incidents))
	for _, inc := range s.incidents {
		list = append(list, inc)
	}
	return list
}

func (s *Store) Advance(id string, stage Stage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inc, ok := s.incidents[id]; ok {
		inc.Stage = stage
		inc.UpdatedAt = time.Now()
	}
}

func (s *Store) Update(id string, fn func(*Incident)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inc, ok := s.incidents[id]; ok {
		fn(inc)
		inc.UpdatedAt = time.Now()
	}
}

func (s *Store) Approve(id string) bool {
	s.mu.RLock()
	inc, ok := s.incidents[id]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case inc.approveCh <- true:
		return true
	default:
		return false
	}
}

func (s *Store) WaitForApproval(id string) bool {
	s.mu.RLock()
	inc, ok := s.incidents[id]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case approved := <-inc.approveCh:
		return approved
	case <-time.After(10 * time.Minute):
		return false
	}
}
