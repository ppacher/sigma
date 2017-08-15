package sigma

import (
	"github.com/homebot/core/utils"
	"github.com/homebot/protobuf/pkg/api/sigma"
)

// TriggerSpec describes a sigma function trigger
type TriggerSpec struct {
	// Type is the type of trigger to build
	Type string `json:"type" yaml:"type"`

	// Condition holds a govaluate expression that is evaluated against the
	// event before the function is triggered
	Condition string `json:"when" yaml:"when"`

	// Options holds additional options for building the trigger
	Options map[string]string `json:"options" yaml:"options"`
}

// ToProtobuf converts the trigger spec to it's protocol buffer representation
func (t TriggerSpec) ToProtobuf() *sigma.TriggerSpec {
	return &sigma.TriggerSpec{
		Type:      t.Type,
		Condition: t.Condition,
		Options:   t.Options,
	}
}

// TriggerSpecFromProtobuf creates a trigger spec from it's protocol buffer
// representation
func TriggerSpecFromProtobuf(t *sigma.TriggerSpec) TriggerSpec {
	return TriggerSpec{
		Type:      t.GetType(),
		Options:   t.GetOptions(),
		Condition: t.GetCondition(),
	}
}

// FunctionSpec describes a function to be executed and managed by funker
type FunctionSpec struct {
	// ID holds the ID of the function specification
	ID string `json:"id" yaml:"id"`

	// Type is the type of function and is used to select the node type
	Type string `json:"type" yaml:"type"`

	// Content holds the content of the function. The content type depends on
	// the node executor
	Content string `json:"content" yaml:"content"`

	// Policies are auto-scaling policies for the function
	Policies map[string]map[string]string `json:"policies" yaml:"policies"`

	// Triggers holds trigger specifications for the function
	Triggers []TriggerSpec `json:"triggers" yaml:"triggers"`

	// Parameters may hold optional parameters for the function
	Parameteres utils.ValueMap `json:"parameters" yaml:"parameters"`
}

// TriggersToProtobuf converts a slice or array of triggers to their
// protocol buffer representation
func TriggersToProtobuf(t []TriggerSpec) []*sigma.TriggerSpec {
	var res []*sigma.TriggerSpec

	for _, trigger := range t {
		res = append(res, trigger.ToProtobuf())
	}

	return res
}

// TriggersFromProtobuf converts a slice or array of protocol buffer
// triggers to a slice of TriggerSpec
func TriggersFromProtobuf(t []*sigma.TriggerSpec) []TriggerSpec {
	var res []TriggerSpec

	for _, trigger := range t {
		res = append(res, TriggerSpecFromProtobuf(trigger))
	}

	return res
}

// PoliciesToProtobuf creates a list of protocol buffer policy definitions
func PoliciesToProtobuf(policies map[string]map[string]string) []*sigma.Policy {
	var p []*sigma.Policy

	for name, opts := range policies {
		p = append(p, &sigma.Policy{
			Type:    name,
			Options: opts,
		})
	}

	return p
}

// ProtobufToPolicies creates a policy configuration map from it's protobuf
// representationG
func ProtobufToPolicies(in []*sigma.Policy) map[string]map[string]string {
	res := make(map[string]map[string]string)

	for _, p := range in {
		res[p.GetName()] = p.GetOptions()
	}

	return res
}

// ToProtobuf converts the function spec to it's protocol buffer representation
func (spec FunctionSpec) ToProtobuf() *sigma.FunctionSpec {
	return &sigma.FunctionSpec{
		Id:         spec.ID,
		Type:       spec.Type,
		Policies:   PoliciesToProtobuf(spec.Policies),
		Content:    []byte(spec.Content),
		Triggers:   TriggersToProtobuf(spec.Triggers),
		Parameters: spec.Parameteres.ToProto(),
	}
}

// SpecFromProto creates a function spec from it's protocol buffer
// representation
func SpecFromProto(in *sigma.FunctionSpec) FunctionSpec {
	return FunctionSpec{
		ID:          in.GetId(),
		Type:        in.GetType(),
		Policies:    ProtobufToPolicies(in.GetPolicies()),
		Content:     string(in.GetContent()),
		Triggers:    TriggersFromProtobuf(in.GetTriggers()),
		Parameteres: utils.ValueMapFrom(in.GetParameters()),
	}
}
