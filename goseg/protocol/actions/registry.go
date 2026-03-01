package actions

import "fmt"

type ActionBinding[A ~string, S any, C any] struct {
	Action    A
	Execute   func(S, C) error
	Operation func(C) string
}

type ActionDispatcher[A ~string, S any, C any] struct {
	namespace Namespace
	order     []A
	bindings  map[A]ActionBinding[A, S, C]
}

func NewActionDispatcher[A ~string, S any, C any](namespace Namespace, bindings []ActionBinding[A, S, C]) ActionDispatcher[A, S, C] {
	bindingMap := make(map[A]ActionBinding[A, S, C], len(bindings))
	order := make([]A, 0, len(bindings))
	for _, binding := range bindings {
		bindingMap[binding.Action] = binding
		order = append(order, binding.Action)
	}
	return ActionDispatcher[A, S, C]{
		namespace: namespace,
		order:     order,
		bindings:  bindingMap,
	}
}

func (d ActionDispatcher[A, S, C]) Supported() []A {
	supported := make([]A, len(d.order))
	copy(supported, d.order)
	return supported
}

func (d ActionDispatcher[A, S, C]) Execute(action A, svc S, payload C) error {
	binding, ok := d.bindings[action]
	if !ok {
		return UnsupportedActionError{
			Namespace: d.namespace,
			Action:    Action(action),
		}
	}
	return binding.Execute(svc, payload)
}

func (d ActionDispatcher[A, S, C]) Describe(action A, payload C) (string, bool) {
	binding, ok := d.bindings[action]
	if !ok {
		return fmt.Sprintf("%s action %s", d.namespace, action), false
	}
	if binding.Operation == nil {
		return fmt.Sprintf("%s action %s", d.namespace, action), true
	}
	return binding.Operation(payload), true
}
