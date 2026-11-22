package tests

import "testing"

// Chapter 2, Conformance. What an implementation may add is never asserted
// absent; these rows check only what it must provide.

func TestS2_Conformance(t *testing.T) {
	runRows(t, []row{
		// 2: "A conforming implementation of ECMAScript must provide and support all the types, values, objects, properties, functions"
		{"2 every global constructor and function of the spec is provided",
			output(`print(typeof Object, typeof Function, typeof Array, typeof String, typeof Boolean, typeof Number, typeof Date)`+
				`; print(typeof eval, typeof parseInt, typeof parseFloat, typeof escape, typeof unescape, typeof isNaN, typeof isFinite)`+
				`; print(typeof Math, typeof NaN, typeof Infinity)`,
				"function function function function function function function\n"+
					"function function function function function function function\n"+
					"object number number\n")},
		// 2: "must provide and support all the types": Undefined, Null, Boolean, String, Number and Object (§8)
		{"2 every type of the spec has values",
			output(`print(typeof void 0, typeof null, typeof false, typeof "", typeof 0, typeof new Object())`,
				"undefined object boolean string number object\n")},
	})
}
