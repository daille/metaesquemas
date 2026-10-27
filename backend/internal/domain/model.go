// Package domain contiene el modelo del catálogo, sin dependencias de infraestructura.
package domain

import (
	"encoding/json"
	"errors"
)

var (
	ErrNotFound = errors.New("recurso no encontrado")
	ErrInvalid  = errors.New("solicitud inválida")
)

type Responsable struct {
	Nombre   string `json:"nombre"`
	Contacto string `json:"contacto,omitempty"`
	Heredado bool   `json:"heredado"`
	Origen   string `json:"origen,omitempty"`
}

type Vigencia struct {
	Desde    string  `json:"desde"`
	Hasta    *string `json:"hasta"`
	Heredada bool    `json:"heredada"`
}

type Dep struct {
	ID       string `json:"id"`
	Relation string `json:"relation"`
	Exists   bool   `json:"exists"`
}

type Use struct {
	ID       string `json:"id"`
	Relation string `json:"relation"`
}

// Entry es un esquema del catálogo (resumen del índice pre-calculado).
type Entry struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Namespace    string           `json:"namespace"`
	Kind         string           `json:"kind"`
	Tipo         string           `json:"tipo"`
	TipoInferido bool             `json:"tipoInferido"`
	Doc          string           `json:"doc,omitempty"`
	Responsable  *Responsable     `json:"responsable"`
	Vigencia     *Vigencia        `json:"vigencia"`
	Deprecado    *json.RawMessage `json:"deprecado,omitempty"`
	Servicio     *json.RawMessage `json:"servicio,omitempty"`
	Plantilla    *json.RawMessage `json:"plantilla,omitempty"`
	File         string           `json:"file"`
	Version      int              `json:"version"`
	Projections  []string         `json:"projections"`
	Dependencies []Dep            `json:"dependencies"`
	UsedBy       []Use            `json:"usedBy"`

	// Calculados al responder (no vienen del índice)
	VigenciaEstado  string `json:"vigenciaEstado,omitempty"`
	UsedByCount     int    `json:"usedByCount"`
	DependencyCount int    `json:"dependencyCount"`
}

type VersionRef struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"createdAt"`
	Motivo    string `json:"motivo"`
}

type FileInfo struct {
	Path      string       `json:"path"`
	Namespace string       `json:"namespace"`
	Version   int          `json:"version"`
	SHA256    string       `json:"sha256"`
	UpdatedAt string       `json:"updatedAt"`
	Imports   []string     `json:"imports"`
	Versions  []VersionRef `json:"versions"`
}

type Index struct {
	Epoch       int        `json:"epoch"`
	GeneratedAt string     `json:"generatedAt"`
	Files       []FileInfo `json:"files"`
	Entries     []Entry    `json:"entries"`
}

type Snippet struct {
	Text      string `json:"text"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

// Detail es un esquema con todo su detalle, leído de la versión pedida del repositorio.
type Detail struct {
	Entry
	Extends     string            `json:"extends,omitempty"`
	BaseType    string            `json:"baseType,omitempty"`
	Builtin     string            `json:"builtin,omitempty"`
	Constraints json.RawMessage   `json:"constraints,omitempty"`
	Members     json.RawMessage   `json:"members,omitempty"`
	Values      json.RawMessage   `json:"values,omitempty"`
	Variants    json.RawMessage   `json:"variants,omitempty"`
	Operations  json.RawMessage   `json:"operations,omitempty"`
	Source      *Snippet          `json:"source,omitempty"`
	Available   map[string]string `json:"available"` // proyección -> URL
	Versions    []VersionRef      `json:"versions"`
	Requested   int               `json:"requestedVersion"`
	IsCurrent   bool              `json:"isCurrent"`
	Build       json.RawMessage   `json:"build,omitempty"`
	FileInfo    *FileInfo         `json:"fileInfo,omitempty"`
}

type Stats struct {
	Total        int            `json:"total"`
	ByTipo       map[string]int `json:"byTipo"`
	Vigentes     int            `json:"vigentes"`
	Vencidos     int            `json:"vencidos"`
	Archivos     int            `json:"archivos"`
	Responsables int            `json:"responsables"`
	MasUsados    []Entry        `json:"masUsados"`
	Actualizado  string         `json:"actualizado"`
}

type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type Facets struct {
	Tipos        []Facet `json:"tipos"`
	Responsables []Facet `json:"responsables"`
	Vigencia     []Facet `json:"vigencia"`
	Usada        []Facet `json:"usada"`
}

type NsNode struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Count    int       `json:"count"` // esquemas en este namespace y sus subnamespaces
	Children []*NsNode `json:"children,omitempty"`
}

type ListParams struct {
	Q, Namespace, Responsable, Vigencia, Sort string
	Tipos                                     []string
	Usada                                     string // "", "si", "no"
	Limit, Offset                             int
}

type ListResult struct {
	Items  []Entry `json:"items"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

type TreeNode struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Tipo      string      `json:"tipo,omitempty"`
	Relation  string      `json:"relation,omitempty"`  // relación principal
	Relations []string    `json:"relations,omitempty"` // todas las relaciones con el nodo padre
	Exists    bool        `json:"exists"`
	Cycle     bool        `json:"cycle,omitempty"`
	Children  []*TreeNode `json:"children,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}

type GraphNode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Tipo      string `json:"tipo"`
	Namespace string `json:"namespace"`
	Exists    bool   `json:"exists"`
}

type GraphEdge struct {
	Source    string   `json:"source"`
	Target    string   `json:"target"`
	Relation  string   `json:"relation"`  // relación principal (define el estilo)
	Relations []string `json:"relations"` // todas las relaciones entre ambos esquemas
}

type Graph struct {
	Center string      `json:"center"`
	Nodes  []GraphNode `json:"nodes"`
	Edges  []GraphEdge `json:"edges"`
}
