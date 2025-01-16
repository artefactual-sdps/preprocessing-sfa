// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db/predicate"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db/sip"
)

// SIPUpdate is the builder for updating SIP entities.
type SIPUpdate struct {
	config
	hooks    []Hook
	mutation *SIPMutation
}

// Where appends a list predicates to the SIPUpdate builder.
func (su *SIPUpdate) Where(ps ...predicate.SIP) *SIPUpdate {
	su.mutation.Where(ps...)
	return su
}

// SetName sets the "name" field.
func (su *SIPUpdate) SetName(s string) *SIPUpdate {
	su.mutation.SetName(s)
	return su
}

// SetNillableName sets the "name" field if the given value is not nil.
func (su *SIPUpdate) SetNillableName(s *string) *SIPUpdate {
	if s != nil {
		su.SetName(*s)
	}
	return su
}

// SetChecksum sets the "checksum" field.
func (su *SIPUpdate) SetChecksum(s string) *SIPUpdate {
	su.mutation.SetChecksum(s)
	return su
}

// SetNillableChecksum sets the "checksum" field if the given value is not nil.
func (su *SIPUpdate) SetNillableChecksum(s *string) *SIPUpdate {
	if s != nil {
		su.SetChecksum(*s)
	}
	return su
}

// Mutation returns the SIPMutation object of the builder.
func (su *SIPUpdate) Mutation() *SIPMutation {
	return su.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (su *SIPUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, su.sqlSave, su.mutation, su.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (su *SIPUpdate) SaveX(ctx context.Context) int {
	affected, err := su.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (su *SIPUpdate) Exec(ctx context.Context) error {
	_, err := su.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (su *SIPUpdate) ExecX(ctx context.Context) {
	if err := su.Exec(ctx); err != nil {
		panic(err)
	}
}

func (su *SIPUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(sip.Table, sip.Columns, sqlgraph.NewFieldSpec(sip.FieldID, field.TypeInt))
	if ps := su.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := su.mutation.Name(); ok {
		_spec.SetField(sip.FieldName, field.TypeString, value)
	}
	if value, ok := su.mutation.Checksum(); ok {
		_spec.SetField(sip.FieldChecksum, field.TypeString, value)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, su.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{sip.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	su.mutation.done = true
	return n, nil
}

// SIPUpdateOne is the builder for updating a single SIP entity.
type SIPUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *SIPMutation
}

// SetName sets the "name" field.
func (suo *SIPUpdateOne) SetName(s string) *SIPUpdateOne {
	suo.mutation.SetName(s)
	return suo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (suo *SIPUpdateOne) SetNillableName(s *string) *SIPUpdateOne {
	if s != nil {
		suo.SetName(*s)
	}
	return suo
}

// SetChecksum sets the "checksum" field.
func (suo *SIPUpdateOne) SetChecksum(s string) *SIPUpdateOne {
	suo.mutation.SetChecksum(s)
	return suo
}

// SetNillableChecksum sets the "checksum" field if the given value is not nil.
func (suo *SIPUpdateOne) SetNillableChecksum(s *string) *SIPUpdateOne {
	if s != nil {
		suo.SetChecksum(*s)
	}
	return suo
}

// Mutation returns the SIPMutation object of the builder.
func (suo *SIPUpdateOne) Mutation() *SIPMutation {
	return suo.mutation
}

// Where appends a list predicates to the SIPUpdate builder.
func (suo *SIPUpdateOne) Where(ps ...predicate.SIP) *SIPUpdateOne {
	suo.mutation.Where(ps...)
	return suo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (suo *SIPUpdateOne) Select(field string, fields ...string) *SIPUpdateOne {
	suo.fields = append([]string{field}, fields...)
	return suo
}

// Save executes the query and returns the updated SIP entity.
func (suo *SIPUpdateOne) Save(ctx context.Context) (*SIP, error) {
	return withHooks(ctx, suo.sqlSave, suo.mutation, suo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (suo *SIPUpdateOne) SaveX(ctx context.Context) *SIP {
	node, err := suo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (suo *SIPUpdateOne) Exec(ctx context.Context) error {
	_, err := suo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (suo *SIPUpdateOne) ExecX(ctx context.Context) {
	if err := suo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (suo *SIPUpdateOne) sqlSave(ctx context.Context) (_node *SIP, err error) {
	_spec := sqlgraph.NewUpdateSpec(sip.Table, sip.Columns, sqlgraph.NewFieldSpec(sip.FieldID, field.TypeInt))
	id, ok := suo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`db: missing "SIP.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := suo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, sip.FieldID)
		for _, f := range fields {
			if !sip.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != sip.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := suo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := suo.mutation.Name(); ok {
		_spec.SetField(sip.FieldName, field.TypeString, value)
	}
	if value, ok := suo.mutation.Checksum(); ok {
		_spec.SetField(sip.FieldChecksum, field.TypeString, value)
	}
	_node = &SIP{config: suo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, suo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{sip.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	suo.mutation.done = true
	return _node, nil
}
