// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package ast

import "fmt"

// Transformer defines the interface for transforming AST elements. If the
// transformer returns nil and does not indicate an error, the AST element will
// be set to nil and no transformations will be applied to children of the
// element.
type Transformer interface {
	Transform(v interface{}) (interface{}, error)
}

// Transform iterates the AST and calls the Transform function on the
// Transformer t for x before recursing.
func Transform(t Transformer, x interface{}) (interface{}, error) {

	if term, ok := x.(*Term); ok {
		return Transform(t, term.Value)
	}

	y, err := t.Transform(x)
	if err != nil {
		return x, err
	}

	if y == nil {
		return nil, nil
	}

	var ok bool
	switch y := y.(type) {
	case *Module:
		p, err := Transform(t, y.Package)
		if err != nil {
			return nil, err
		}
		if y.Package, ok = p.(*Package); !ok {
			return nil, fmt.Errorf("illegal transform: %T != %T", y.Package, p)
		}
		for i := range y.Imports {
			imp, err := Transform(t, y.Imports[i])
			if err != nil {
				return nil, err
			}
			if y.Imports[i], ok = imp.(*Import); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y.Imports[i], imp)
			}
		}
		for i := range y.Rules {
			rule, err := Transform(t, y.Rules[i])
			if err != nil {
				return nil, err
			}
			if y.Rules[i], ok = rule.(*Rule); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y.Rules[i], rule)
			}
		}
		for i := range y.Comments {
			comment, err := Transform(t, y.Comments[i])
			if err != nil {
				return nil, err
			}
			if y.Comments[i], ok = comment.(*Comment); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y.Comments[i], comment)
			}
		}
		return y, nil
	case *Package:
		ref, err := Transform(t, y.Path)
		if err != nil {
			return nil, err
		}
		if y.Path, ok = ref.(Ref); !ok {
			return nil, fmt.Errorf("illegal transform: %T != %T", y.Path, ref)
		}
		return y, nil
	case *Import:
		y.Path, err = transformTerm(t, y.Path)
		if err != nil {
			return nil, err
		}
		if y.Alias, err = transformVar(t, y.Alias); err != nil {
			return nil, err
		}
		return y, nil
	case *Rule:
		if y.Head, err = transformHead(t, y.Head); err != nil {
			return nil, err
		}
		if y.Body, err = transformBody(t, y.Body); err != nil {
			return nil, err
		}
		if y.Else != nil {
			rule, err := Transform(t, y.Else)
			if err != nil {
				return nil, err
			}
			if y.Else, ok = rule.(*Rule); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y.Else, rule)
			}
		}
		return y, nil
	case *Head:
		if y.Name, err = transformVar(t, y.Name); err != nil {
			return nil, err
		}
		if y.Key != nil {
			if y.Key, err = transformTerm(t, y.Key); err != nil {
				return nil, err
			}
		}
		if y.Value != nil {
			if y.Value, err = transformTerm(t, y.Value); err != nil {
				return nil, err
			}
		}
		return y, nil
	case *Func:
		if y.Head, err = transformFuncHead(t, y.Head); err != nil {
			return nil, err
		}
		if y.Body, err = transformBody(t, y.Body); err != nil {
			return nil, err
		}
		return y, nil
	case *FuncHead:
		if y.Name, err = transformVar(t, y.Name); err != nil {
			return nil, err
		}
		for i := range y.Args {
			if y.Args[i], err = transformTerm(t, y.Args[i]); err != nil {
				return nil, err
			}
		}
		if y.Output, err = transformTerm(t, y.Output); err != nil {
			return nil, err
		}
		return y, nil
	case Args:
		for i := range y {
			if y[i], err = transformTerm(t, y[i]); err != nil {
				return nil, err
			}
		}
		return y, nil
	case Body:
		for i, e := range y {
			e, err := Transform(t, e)
			if err != nil {
				return nil, err
			}
			if y[i], ok = e.(*Expr); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y[i], e)
			}
		}
		return y, nil
	case *Expr:
		switch ts := y.Terms.(type) {
		case []*Term:
			for i := range ts {
				if ts[i], err = transformTerm(t, ts[i]); err != nil {
					return nil, err
				}
			}
		case *Term:
			if y.Terms, err = transformTerm(t, ts); err != nil {
				return nil, err
			}
		}
		for i, w := range y.With {
			w, err := Transform(t, w)
			if err != nil {
				return nil, err
			}
			if y.With[i], ok = w.(*With); !ok {
				return nil, fmt.Errorf("illegal transform: %T != %T", y.With[i], w)
			}
		}
		return y, nil
	case *With:
		if y.Target, err = transformTerm(t, y.Target); err != nil {
			return nil, err
		}
		if y.Value, err = transformTerm(t, y.Value); err != nil {
			return nil, err
		}
		return y, nil
	case Ref:
		for i, term := range y {
			if y[i], err = transformTerm(t, term); err != nil {
				return nil, err
			}
		}
		return y, nil
	case Object:
		for i, elem := range y {
			k, err := transformTerm(t, elem[0])
			if err != nil {
				return nil, err
			}
			v, err := transformTerm(t, elem[1])
			if err != nil {
				return nil, err
			}
			y[i] = Item(k, v)
		}
		return y, nil
	case Array:
		for i := range y {
			if y[i], err = transformTerm(t, y[i]); err != nil {
				return nil, err
			}
		}
		return y, nil
	case *Set:
		y, err = y.Map(func(term *Term) (*Term, error) {
			return transformTerm(t, term)
		})
		if err != nil {
			return nil, err
		}
		return y, nil
	case *ArrayComprehension:
		if y.Term, err = transformTerm(t, y.Term); err != nil {
			return nil, err
		}
		if y.Body, err = transformBody(t, y.Body); err != nil {
			return nil, err
		}
		return y, nil
	case *ObjectComprehension:
		if y.Key, err = transformTerm(t, y.Key); err != nil {
			return nil, err
		}
		if y.Value, err = transformTerm(t, y.Value); err != nil {
			return nil, err
		}
		if y.Body, err = transformBody(t, y.Body); err != nil {
			return nil, err
		}
		return y, nil
	case *SetComprehension:
		if y.Term, err = transformTerm(t, y.Term); err != nil {
			return nil, err
		}
		if y.Body, err = transformBody(t, y.Body); err != nil {
			return nil, err
		}
		return y, nil
	default:
		return y, nil
	}
}

// TransformRefs calls the function f on all references under x.
func TransformRefs(x interface{}, f func(Ref) (Value, error)) (interface{}, error) {
	t := &GenericTransformer{func(x interface{}) (interface{}, error) {
		if r, ok := x.(Ref); ok {
			return f(r)
		}
		return x, nil
	}}
	return Transform(t, x)
}

// GenericTransformer implements the Transformer interface to provide a utility
// to transform AST nodes using a closure.
type GenericTransformer struct {
	f func(x interface{}) (interface{}, error)
}

// Transform calls the function f on the GenericTransformer.
func (t *GenericTransformer) Transform(x interface{}) (interface{}, error) {
	return t.f(x)
}

func transformHead(t Transformer, head *Head) (*Head, error) {
	y, err := Transform(t, head)
	if err != nil {
		return nil, err
	}
	h, ok := y.(*Head)
	if !ok {
		return nil, fmt.Errorf("illegal transform: %T != %T", head, y)
	}
	return h, nil
}

func transformFuncHead(t Transformer, head *FuncHead) (*FuncHead, error) {
	y, err := Transform(t, head)
	if err != nil {
		return nil, err
	}
	h, ok := y.(*FuncHead)
	if !ok {
		return nil, fmt.Errorf("illegal transform: %T != %T", head, y)
	}
	return h, nil
}

func transformBody(t Transformer, body Body) (Body, error) {
	y, err := Transform(t, body)
	if err != nil {
		return nil, err
	}
	r, ok := y.(Body)
	if !ok {
		return nil, fmt.Errorf("illegal transform: %T != %T", body, y)
	}
	return r, nil
}

func transformTerm(t Transformer, term *Term) (*Term, error) {
	v, err := transformValue(t, term.Value)
	if err != nil {
		return nil, err
	}
	r := &Term{
		Value:    v,
		Location: term.Location,
	}
	return r, nil
}

func transformValue(t Transformer, v Value) (Value, error) {
	v1, err := Transform(t, v)
	if err != nil {
		return nil, err
	}
	r, ok := v1.(Value)
	if !ok {
		return nil, fmt.Errorf("illegal transform: %T != %T", v, v1)
	}
	return r, nil
}

func transformVar(t Transformer, v Var) (Var, error) {
	v1, err := Transform(t, v)
	if err != nil {
		return "", err
	}
	r, ok := v1.(Var)
	if !ok {
		return "", fmt.Errorf("illegal transform: %T != %T", v, v1)
	}
	return r, nil
}
