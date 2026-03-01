package forge

import (
	"context"
	"database/sql"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// DB is satisfied by *sql.DB, *sql.Tx, and any pgx adapter such as
// forgepgx.Wrap(pool). Users pass a concrete implementation to
// forge.Config — they do not implement DB directly.
type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// dbField maps a SQL column name to the index of the matching struct field.
type dbField struct {
	index int
	name  string // column name: db tag value, or lowercased field name
}

// dbFieldCache stores []dbField slices keyed by reflect.Type (struct, not pointer).
// Entries are written once per type and never modified.
var dbFieldCache sync.Map

// dbFields returns the cached column→field mapping for struct type t.
// On cache miss, it iterates the exported fields and stores the result.
func dbFields(t reflect.Type) []dbField {
	if v, ok := dbFieldCache.Load(t); ok {
		return v.([]dbField)
	}
	fields := make([]dbField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("db")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		fields = append(fields, dbField{index: i, name: name})
	}
	dbFieldCache.Store(t, fields)
	return fields
}

// goFieldCache stores map[string]int (Go field name → index) keyed by reflect.Type.
// Used by MemoryRepo to locate fields such as ID and Slug by their Go names.
var goFieldCache sync.Map

// goFields returns a cached Go-name → field-index map for struct type t.
// Only maps direct (non-embedded) exported fields.
func goFields(t reflect.Type) map[string]int {
	if v, ok := goFieldCache.Load(t); ok {
		return v.(map[string]int)
	}
	m := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if f := t.Field(i); f.IsExported() {
			m[f.Name] = i
		}
	}
	goFieldCache.Store(t, m)
	return m
}

// goFieldPathKey is the cache key for [goFieldPath].
type goFieldPathKey struct {
	t    reflect.Type
	name string
}

// goFieldPathCache stores field index paths ([]int) keyed by goFieldPathKey.
// A []int path is used to access promoted fields in embedded structs via
// reflect.Value.FieldByIndex.
var goFieldPathCache sync.Map

// goFieldPath returns the reflect index path (suitable for FieldByIndex) for
// the named field in struct type t, traversing embedded structs. Returns nil
// if the field is not found.
func goFieldPath(t reflect.Type, name string) []int {
	key := goFieldPathKey{t: t, name: name}
	if v, ok := goFieldPathCache.Load(key); ok {
		return v.([]int)
	}
	sf, ok := t.FieldByName(name)
	if !ok {
		goFieldPathCache.Store(key, ([]int)(nil))
		return nil
	}
	path := sf.Index
	goFieldPathCache.Store(key, path)
	return path
}

// structTypeOf returns the underlying struct reflect.Type for T and whether T
// is a pointer type. Panics if T resolves to neither a struct nor a pointer
// to a struct — Query[T] is only valid for struct targets.
func structTypeOf[T any]() (reflect.Type, bool) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		return t.Elem(), true
	}
	return t, false
}

// Query executes a SQL query and scans the result rows into a slice of T.
// T may be a struct type or a pointer to a struct (e.g. *BlogPost).
// Columns are matched to fields by db struct tag first, then by lowercased
// field name. Unrecognised columns are discarded without error.
// Returns an empty (non-nil) slice when no rows match.
func Query[T any](ctx context.Context, db DB, query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	elemType, isPtr := structTypeOf[T]()
	fields := dbFields(elemType)

	// Build column-name → field-index lookup from the cached field list.
	colIdx := make(map[string]int, len(fields))
	for _, f := range fields {
		colIdx[f.name] = f.index
	}

	result := make([]T, 0)
	for rows.Next() {
		ptr := reflect.New(elemType)
		elem := ptr.Elem()

		targets := make([]any, len(cols))
		for i, col := range cols {
			if idx, ok := colIdx[col]; ok {
				targets[i] = elem.Field(idx).Addr().Interface()
			} else {
				var discard any
				targets[i] = &discard
			}
		}

		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}

		var item T
		if isPtr {
			item = ptr.Interface().(T)
		} else {
			item = elem.Interface().(T)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// QueryOne executes a SQL query and returns the first scanned row as T.
// Returns ErrNotFound when no rows match.
func QueryOne[T any](ctx context.Context, db DB, query string, args ...any) (T, error) {
	rows, err := Query[T](ctx, db, query, args...)
	if err != nil {
		var zero T
		return zero, err
	}
	if len(rows) == 0 {
		var zero T
		return zero, ErrNotFound
	}
	return rows[0], nil
}

// ListOptions controls pagination and ordering for FindAll queries.
type ListOptions struct {
	// Page is one-based. Values ≤ 0 are treated as page 1.
	Page int
	// PerPage is the maximum number of items per page.
	// A value of 0 means return all items.
	PerPage int
	// OrderBy is the Go field name to sort by (e.g. "Title").
	// Sorting applies only to exported string fields; other types are ignored.
	OrderBy string
	// Desc reverses the sort order when true.
	Desc bool
}

// Offset returns the zero-based row offset for the page described by o.
func (o ListOptions) Offset() int {
	p := o.Page
	if p <= 0 {
		p = 1
	}
	off := (p - 1) * o.PerPage
	if off < 0 {
		return 0
	}
	return off
}

// Repository is the storage interface for a content type.
// Implement it to provide a custom storage backend.
// Use [NewMemoryRepo] for in-process testing and prototyping.
type Repository[T any] interface {
	FindByID(ctx context.Context, id string) (T, error)
	FindBySlug(ctx context.Context, slug string) (T, error)
	FindAll(ctx context.Context, opts ListOptions) ([]T, error)
	Save(ctx context.Context, node T) error
	Delete(ctx context.Context, id string) error
}

// MemoryRepo is a thread-safe in-memory implementation of [Repository].
// It is intended for unit tests and prototyping — not production use.
// Fields named ID and Slug are located via cached reflection on first use.
type MemoryRepo[T any] struct {
	mu    sync.RWMutex
	items map[string]T
	order []string // ID insertion order
}

// NewMemoryRepo returns an empty MemoryRepo[T] ready for use.
func NewMemoryRepo[T any]() *MemoryRepo[T] {
	return &MemoryRepo[T]{items: make(map[string]T)}
}

// Save upserts node into the repository keyed by its ID field.
// On insert the ID is appended to the internal order list.
// On update the existing position in the order list is preserved.
func (r *MemoryRepo[T]) Save(_ context.Context, node T) error {
	id := stringField(node, "ID")
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[id]; !exists {
		r.order = append(r.order, id)
	}
	r.items[id] = node
	return nil
}

// FindByID returns the item with the given ID, or ErrNotFound.
func (r *MemoryRepo[T]) FindByID(_ context.Context, id string) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		var zero T
		return zero, ErrNotFound
	}
	return item, nil
}

// FindBySlug returns the first item whose Slug field matches slug, or ErrNotFound.
func (r *MemoryRepo[T]) FindBySlug(_ context.Context, slug string) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, id := range r.order {
		item := r.items[id]
		if stringField(item, "Slug") == slug {
			return item, nil
		}
	}
	var zero T
	return zero, ErrNotFound
}

// FindAll returns items in insertion order, with optional sorting and
// pagination from opts. When opts.PerPage is 0, all items are returned.
func (r *MemoryRepo[T]) FindAll(_ context.Context, opts ListOptions) ([]T, error) {
	r.mu.RLock()
	all := make([]T, 0, len(r.order))
	for _, id := range r.order {
		all = append(all, r.items[id])
	}
	r.mu.RUnlock()

	if opts.OrderBy != "" {
		sortItems(all, opts.OrderBy, opts.Desc)
	}

	if opts.PerPage <= 0 {
		return all, nil
	}

	off := opts.Offset()
	if off >= len(all) {
		return []T{}, nil
	}
	end := off + opts.PerPage
	if end > len(all) {
		end = len(all)
	}
	return all[off:end], nil
}

// Delete removes the item with the given ID. Returns ErrNotFound if absent.
func (r *MemoryRepo[T]) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	for i, oid := range r.order {
		if oid == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

// stringField returns the string value of the named exported field in v.
// Traverses one pointer indirection and handles embedded struct fields.
// Returns "" if the field does not exist, is not a string, or v is nil.
// Uses goFieldPathCache to avoid repeated reflection.
func stringField[T any](v T, name string) string {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	path := goFieldPath(rv.Type(), name)
	if path == nil {
		return ""
	}
	f := rv.FieldByIndex(path)
	if f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

// sortItems sorts items in-place by the named string field.
// Non-string fields and missing fields sort to the end.
// Uses a stable sort so equal elements preserve insertion order.
func sortItems[T any](items []T, field string, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a := stringField(items[i], field)
		b := stringField(items[j], field)
		if desc {
			return a > b
		}
		return a < b
	})
}
