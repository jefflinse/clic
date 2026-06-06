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

	// scalar fields (and scalar arrays) are gathered in one form; arrays whose
	// elements are objects can't be a single input, so they are collected after
	// via a repeatable sub-form.
	inputs := []huh.Field{}
	for _, b := range bindings {
		inputs = append(inputs, b.inputs("")...)
	}
	if len(inputs) > 0 {
		if err := huh.NewForm(huh.NewGroup(inputs...)).Run(); err != nil {
			return nil, err
		}
	}

	if err := collectComplexArrays(bindings); err != nil {
		return nil, err
	}

	return assemble(bindings), nil
}

// binding pairs a field with the holder variables huh writes into, plus the
// child bindings of a nested object and the collected entries of an object array.
type binding struct {
	field    form.Field
	str      string
	boolean  bool
	children []*binding
	elements []any
}

// isComplexArray reports whether a field is an array whose element type cannot
// be entered as a single line of text (i.e. an object or nested array).
func isComplexArray(f form.Field) bool {
	return f.Type == form.ArrayField && f.Item != nil &&
		(f.Item.Type == form.ObjectField || f.Item.Type == form.ArrayField)
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

// inputs returns the huh field(s) this binding contributes to the form. The
// prefix qualifies nested labels with their parent path (e.g. "category.id") so
// fields sharing a name across nesting levels stay distinguishable.
func (b *binding) inputs(prefix string) []huh.Field {
	label := b.field.Label()
	if prefix != "" {
		label = prefix + "." + label
	}
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
		if isComplexArray(b.field) {
			// collected after the main form via a repeatable sub-form
			return nil
		}
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
			inputs = append(inputs, child.inputs(label)...)
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
		if isComplexArray(b.field) {
			return b.elements
		}
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
	case form.ArrayField:
		if isComplexArray(b.field) {
			return len(b.elements) == 0
		}
		return strings.TrimSpace(b.str) == ""
	default:
		return strings.TrimSpace(b.str) == ""
	}
}

// collectComplexArrays walks the binding tree and, for each object-array field,
// runs a repeatable sub-form to gather its entries. It descends into objects so
// nested object arrays are collected too.
func collectComplexArrays(bindings []*binding) error {
	for _, b := range bindings {
		switch {
		case b.field.Type == form.ObjectField:
			if err := collectComplexArrays(b.children); err != nil {
				return err
			}
		case isComplexArray(b.field):
			elements, err := promptElements(b.field)
			if err != nil {
				return err
			}
			b.elements = elements
		}
	}
	return nil
}

// promptElements repeatedly asks whether to add another entry to an object
// array and, for each, runs a sub-form for the element's fields.
func promptElements(field form.Field) ([]any, error) {
	elements := []any{}
	for {
		add := false
		prompt := fmt.Sprintf("Add an entry to %q?", field.Name)
		if len(elements) > 0 {
			prompt = fmt.Sprintf("Add another entry to %q? (%d so far)", field.Name, len(elements))
		}
		if err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(prompt).Value(&add))).Run(); err != nil {
			return nil, err
		}
		if !add {
			return elements, nil
		}

		elem := newBinding(*field.Item)
		if sub := elem.inputs(field.Name); len(sub) > 0 {
			if err := huh.NewForm(huh.NewGroup(sub...)).Run(); err != nil {
				return nil, err
			}
		}
		elements = append(elements, elem.value())
	}
}
