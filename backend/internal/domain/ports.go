package domain

import (
	"context"
	"io"
	"time"
)

// Catalog es el puerto de lectura del repositorio de esquemas pre-calculados.
// Ninguna operación compila nada: solo lee lo que el worker dejó publicado.
type Catalog interface {
	Stats() Stats
	Facets() Facets
	Namespaces() []*NsNode
	List(p ListParams) ListResult
	Get(id string, version int) (*Detail, error)
	// Artifact abre una proyección ("tsp", "tsp-file", "jsonschema", "xsd", "proto", "openapi").
	Artifact(id, kind string, version int) (Artifact, error)
	Tree(id, direction string, depth int) (*TreeNode, error)
	Graph(id string, depth int) (*Graph, error)
}

type Artifact struct {
	Reader      io.ReadSeeker
	Close       func() error
	ContentType string
	FileName    string
	ModTime     time.Time
	ETag        string
	Immutable   bool
}

// Ingest es el puerto de escritura hacia el worker (cola de trabajos).
type Ingest interface {
	Submit(ctx context.Context, fileName, uploader string, source []byte) (jobID string, err error)
	Job(id string) ([]byte, error)
}
