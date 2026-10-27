package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/org/metaesquemas/internal/domain"
)

var jobIDRe = regexp.MustCompile(`^[0-9]{17}-[0-9a-f]{6}$`)

// Queue implementa el puerto Ingest: deja trabajos en la cola que consume el worker
// (mismo protocolo que worker/src/submit.js). La API nunca compila nada.
type Queue struct{ inbox, jobs string }

func NewQueue(inbox, jobs string) *Queue { return &Queue{inbox: inbox, jobs: jobs} }

func (q *Queue) Submit(_ context.Context, fileName, uploader string, source []byte) (string, error) {
	rnd := make([]byte, 3)
	if _, err := rand.Read(rnd); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	id := now.Format("20060102150405") + fmt.Sprintf("%03d", now.Nanosecond()/1e6) + "-" + hex.EncodeToString(rnd)
	tmp := filepath.Join(q.inbox, ".tmp-"+id)
	if err := os.MkdirAll(tmp, 0o750); err != nil {
		return "", err
	}
	job, _ := json.Marshal(map[string]any{"id": id, "fileName": fileName, "uploader": nilIfEmpty(uploader), "submittedAt": now.Format(time.RFC3339)})
	if err := os.WriteFile(filepath.Join(tmp, "source.tsp"), source, 0o640); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmp, "job.json"), job, 0o640); err != nil {
		return "", err
	}
	if err := os.MkdirAll(q.jobs, 0o750); err != nil {
		return "", err
	}
	st, _ := json.Marshal(map[string]any{"id": id, "status": "queued", "fileName": fileName, "submittedAt": now.Format(time.RFC3339)})
	if err := os.WriteFile(filepath.Join(q.jobs, id+".json"), st, 0o640); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, filepath.Join(q.inbox, id)); err != nil { // visible para el worker solo cuando está completo
		return "", err
	}
	return id, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (q *Queue) Job(id string) ([]byte, error) {
	if !jobIDRe.MatchString(id) {
		return nil, domain.ErrInvalid
	}
	b, err := os.ReadFile(filepath.Join(q.jobs, id+".json"))
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return b, nil
}
