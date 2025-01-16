package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// SIP holds the schema definition for the SIP entity.
type SIP struct {
	ent.Schema
}

// Annotations of the SIP.
func (SIP) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sip"},
	}
}

// Fields of the SIP.
func (SIP) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Annotations(entsql.Annotation{
				Size: 1024,
			}),
		field.String("checksum").
			Annotations(entsql.Annotation{
				Size: 64,
			}).
			Unique(),
	}
}
