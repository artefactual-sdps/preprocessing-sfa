// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db/sip"
)

// SIPCreate is the builder for creating a SIP entity.
type SIPCreate struct {
	config
	mutation *SIPMutation
	hooks    []Hook
}

// SetName sets the "name" field.
func (sc *SIPCreate) SetName(s string) *SIPCreate {
	sc.mutation.SetName(s)
	return sc
}

// SetChecksum sets the "checksum" field.
func (sc *SIPCreate) SetChecksum(s string) *SIPCreate {
	sc.mutation.SetChecksum(s)
	return sc
}

// Mutation returns the SIPMutation object of the builder.
func (sc *SIPCreate) Mutation() *SIPMutation {
	return sc.mutation
}

// Save creates the SIP in the database.
func (sc *SIPCreate) Save(ctx context.Context) (*SIP, error) {
	return withHooks(ctx, sc.sqlSave, sc.mutation, sc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (sc *SIPCreate) SaveX(ctx context.Context) *SIP {
	v, err := sc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sc *SIPCreate) Exec(ctx context.Context) error {
	_, err := sc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sc *SIPCreate) ExecX(ctx context.Context) {
	if err := sc.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (sc *SIPCreate) check() error {
	if _, ok := sc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`db: missing required field "SIP.name"`)}
	}
	if _, ok := sc.mutation.Checksum(); !ok {
		return &ValidationError{Name: "checksum", err: errors.New(`db: missing required field "SIP.checksum"`)}
	}
	return nil
}

func (sc *SIPCreate) sqlSave(ctx context.Context) (*SIP, error) {
	if err := sc.check(); err != nil {
		return nil, err
	}
	_node, _spec := sc.createSpec()
	if err := sqlgraph.CreateNode(ctx, sc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	id := _spec.ID.Value.(int64)
	_node.ID = int(id)
	sc.mutation.id = &_node.ID
	sc.mutation.done = true
	return _node, nil
}

func (sc *SIPCreate) createSpec() (*SIP, *sqlgraph.CreateSpec) {
	var (
		_node = &SIP{config: sc.config}
		_spec = sqlgraph.NewCreateSpec(sip.Table, sqlgraph.NewFieldSpec(sip.FieldID, field.TypeInt))
	)
	if value, ok := sc.mutation.Name(); ok {
		_spec.SetField(sip.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := sc.mutation.Checksum(); ok {
		_spec.SetField(sip.FieldChecksum, field.TypeString, value)
		_node.Checksum = value
	}
	return _node, _spec
}

// SIPCreateBulk is the builder for creating many SIP entities in bulk.
type SIPCreateBulk struct {
	config
	err      error
	builders []*SIPCreate
}

// Save creates the SIP entities in the database.
func (scb *SIPCreateBulk) Save(ctx context.Context) ([]*SIP, error) {
	if scb.err != nil {
		return nil, scb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(scb.builders))
	nodes := make([]*SIP, len(scb.builders))
	mutators := make([]Mutator, len(scb.builders))
	for i := range scb.builders {
		func(i int, root context.Context) {
			builder := scb.builders[i]
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*SIPMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, scb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, scb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				if specs[i].ID.Value != nil {
					id := specs[i].ID.Value.(int64)
					nodes[i].ID = int(id)
				}
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, scb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (scb *SIPCreateBulk) SaveX(ctx context.Context) []*SIP {
	v, err := scb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (scb *SIPCreateBulk) Exec(ctx context.Context) error {
	_, err := scb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (scb *SIPCreateBulk) ExecX(ctx context.Context) {
	if err := scb.Exec(ctx); err != nil {
		panic(err)
	}
}
