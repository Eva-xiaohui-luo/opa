// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package ast

import (
	"testing"
)

type testVis struct {
	elems []interface{}
}

func (vis *testVis) Visit(x interface{}) Visitor {
	vis.elems = append(vis.elems, x)
	return vis
}

func TestVisitor(t *testing.T) {

	rule := MustParseModule(`package a.b

import input.x.y as z

t[x] = y { p[x] = {"foo": [y, 2, {"bar": 3}]}; not q[x]; y = [[x, z] | x = "x"; z = "z"]; z = {"foo": [x, z] | x = "x"; z = "z"}; s = {1 | a[i] = "foo"}; count({1, 2, 3}, n) with input.foo.bar as x }

p { false } else { false } else { true }

fn([x, y]) = z { z = "bar"; trim(x, y, z) }`)
	vis := &testVis{}

	Walk(vis, rule)

	/*
		mod
			package
				data.a.b
					term
						data
					term
						a
					term
						b
			import
				term
					input.x.y
						term
							input
						term
							x
						term
							y
						z
			rule
				head
					t
					term
						x
					term
						y
				body
					expr1
						term
							=
						term
							ref1
								term
									p
								term
									x
						term
							object1
								term
									"foo"
								term
									array
										term
											y
										term
											2
										term
											object2
												term
													"bar"
												term
													3
					expr2
						term
							ref2
								term
									q
								term
									x
					expr3
						term
							=
						term
							y
						term
							compr
								term
									array
										term
											x
										term
											z
								body
									expr4
										term
											=
										term
											x
										term
											"x"
									expr5
										term
											=
										term
											z
										term
											"z"
					expr4
						term
							=
						term
							z
						term
							compr
								key
									term
										"foo"
								value
									array
										term
											x
										term
											z
								body
									expr1
										term
											=
										term
											x
										term
											"x"
									expr2
										term
											=
										term
											z
										term
											"z"
					expr5
						term
							=
						term
							s
						term
							compr
								term
									1
								body
									expr1
										term
											=
										term
											ref
												term
													a
												term
													i

										term
											"foo"
					expr6
						term
							count
						term
							set
								term
									1
								term
									2
								term
									3
						term
							n
						with
							term
								input.foo.bar
									term
										input
									term
										foo
									term
										bar
							term
								baz
			rule
				head
					p
					<nil> # not counted
					term
						true
				body
					expr
						term
							false
				rule
					head
						p
						<nil> # not counted
						term
							true
					body
						expr
							term
								false
					rule
						head
							p
							<nil> # not counted
							term
								true
						body
							expr
								term
									true
			func
				head
					fn
					term
						array
							term
								x
							term
								y
					term
						z
				body
					expr1
						term
							=
						term
							z
						term
							"bar"
					expr2
						term
							trim
						term
							x
						term
							y
						term
							z
	*/
	if len(vis.elems) != 216 {
		t.Errorf("Expected exactly 216 elements in AST but got %d: %v", len(vis.elems), vis.elems)
	}
}

func TestWalkVars(t *testing.T) {
	x := MustParseBody(`x = 1; data.abc[2] = y; y[z] = [q | q = 1]`)
	found := NewVarSet()
	WalkVars(x, func(v Var) bool {
		found.Add(v)
		return false
	})
	expected := NewVarSet(Var("x"), Var("data"), Var("y"), Var("z"), Var("q"))
	if !expected.Equal(found) {
		t.Fatalf("Expected %v but got: %v", expected, found)
	}
}

func TestVarVisitor(t *testing.T) {

	tests := []struct {
		stmt     string
		params   VarVisitorParams
		expected string
	}{
		{"data.foo[x] = bar.baz[y]", VarVisitorParams{SkipRefHead: true}, "[x, y]"},
		{"{x: y}", VarVisitorParams{SkipObjectKeys: true}, "[y]"},
		{`foo = [x | data.a[i] = x]`, VarVisitorParams{SkipClosures: true}, "[foo]"},
		{`x = 1; y = 2; z = x + y; count([x, y, z], z)`, VarVisitorParams{}, "[x, y, z]"},
		{"foo with input.bar.baz as qux[corge]", VarVisitorParams{SkipWithTarget: true}, "[foo, qux, corge]"},
	}

	for _, tc := range tests {
		stmt := MustParseStatement(tc.stmt)

		vis := NewVarVisitor().WithParams(tc.params)
		Walk(vis, stmt)

		expected := NewVarSet()
		for _, x := range MustParseTerm(tc.expected).Value.(Array) {
			expected.Add(x.Value.(Var))
		}

		if !vis.Vars().Equal(expected) {
			t.Errorf("For %v w/ %v expected %v but got: %v", stmt, tc.params, expected, vis.Vars())
		}
	}
}
