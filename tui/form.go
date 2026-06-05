// Package tui renders a form.Field tree as an interactive terminal form using
// the charmbracelet huh library. It is the bridge between clic's UI-agnostic
// field description and a concrete, runnable form; the field-to-value assembly
// is kept separate from the rendering so it can be tested without a terminal.
package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jefflinse/clic/form"
)

// PromptBody renders an interactive form for the given fields and returns the
// collected values assembled into a request-body map. Optional fields left
// blank are omitted from the result.
func PromptBody(fields []form.Field) (map[string]any, error) {
	bindings := newBindings(fields)

	inputs := []huh.Field{}
	for _, b := range bindings {
		inputs = append(inputs, b.inputs()...)
	}
	if len(inputs) == 0 {
		return map[string]any{}, nil
	}

	if err := huh.NewForm(huh.NewGroup(inputs...)).Run(); err != nil {
		return nil, err
	}

	return assemble(bindings), nil
}

// binding pairs a field with the holder variables huh writes into, plus the
// child bindings of a nested object.
type binding struct {
	field    form.Field
	str      string
	boolean  bool
	children []*binding
}

func newBindings(fields []form.Field) []*binding {
	bindings := make([]*binding, 0, len(fields))
	for _, f := range fields {
		bindings = append(bindings, newBinding(f))
	}
	return bindings
}

func newBinding(f form.Field) *binding {
	b := &binding{field: f}
	switch def := f.Default.(type) {
	case string:
		b.str = def
	case bool:
		b.boolean = def
	}
	if f.Type == form.ObjectField {
		b.children = newBindings(f.Fields)
	}
	return b
}

// inputs returns the huh field(s) this binding contributes to the form.
func (b *binding) inputs() []huh.Field {
	label := b.field.Label()
	switch b.field.Type {
	case form.BooleanField:
		return []huh.Field{huh.NewConfirm().Title(label).Description(b.field.Description).Value(&b.boolean)}

	case form.EnumField:
		return []huh.Field{
			huh.NewSelect[string]().
				Title(label).
				Description(b.field.Description).
				Options(huh.NewOptions(b.field.Enum...)...).
				Value(&b.str),
		}

	case form.ArrayField:
		return []huh.Field{
			huh.NewText().
				Title(label + " (one per line)").
				Description(b.field.Description).
				Value(&b.str).
				Validate(b.validate),
		}

	case form.ObjectField:
		inputs := []huh.Field{huh.NewNote().Title(label)}
		for _, child := range b.children {
			inputs = append(inputs, child.inputs()...)
		}
		return inputs

	default: // string, integer, number
		return []huh.Field{
			huh.NewInput().
				Title(label).
				Description(b.field.Description).
				Value(&b.str).
				Validate(b.validate),
		}
	}
}

// validate enforces required-ness and numeric parsing for a scalar input.
func (b *binding) validate(s string) error {
	if b.field.Required && strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s is required", b.field.Label())
	}
	if strings.TrimSpace(s) == "" {
		return nil
	}
	switch b.field.Type {
	case form.IntegerField:
		if _, err := strconv.Atoi(strings.TrimSpace(s)); err != nil {
			return fmt.Errorf("must be a whole number")
		}
	case form.NumberField:
		if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err != nil {
			return fmt.Errorf("must be a number")
		}
	}
	return nil
}

// assemble collects the bindings into a body map, omitting optional fields that
// were left empty.
func assemble(bindings []*binding) map[string]any {
	out := map[string]any{}
	for _, b := range bindings {
		if b.skip() {
			continue
		}
		out[b.field.Name] = b.value()
	}
	return out
}

// value converts a binding's collected input into its typed payload value.
func (b *binding) value() any {
	switch b.field.Type {
	case form.BooleanField:
		return b.boolean
	case form.IntegerField:
		n, _ := strconv.Atoi(strings.TrimSpace(b.str))
		return n
	case form.NumberField:
		n, _ := strconv.ParseFloat(strings.TrimSpace(b.str), 64)
		return n
	case form.ObjectField:
		return assemble(b.children)
	case form.ArrayField:
		return b.arrayValue()
	default:
		return b.str
	}
}

// arrayValue splits the multi-line input into typed elements.
func (b *binding) arrayValue() []any {
	out := []any{}
	for line := range strings.SplitSeq(b.str, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, scalar(b.field.Item, line))
	}
	return out
}

// scalar converts a single string into the type described by an element field.
func scalar(item *form.Field, s string) any {
	if item == nil {
		return s
	}
	switch item.Type {
	case form.IntegerField:
		n, _ := strconv.Atoi(s)
		return n
	case form.NumberField:
		n, _ := strconv.ParseFloat(s, 64)
		return n
	case form.BooleanField:
		return strings.EqualFold(s, "true")
	default:
		return s
	}
}

// skip reports whether an optional, empty field should be omitted from the body.
func (b *binding) skip() bool {
	return !b.field.Required && b.empty()
}

// empty reports whether no value was supplied for the field.
func (b *binding) empty() bool {
	switch b.field.Type {
	case form.BooleanField:
		return false
	case form.ObjectField:
		for _, child := range b.children {
			if !child.empty() {
				return false
			}
		}
		return true
	default:
		return strings.TrimSpace(b.str) == ""
	}
}
