package transpile

// Compile renders the project using the named runtime and returns
// the resulting Tree. This is the public entry point for the
// compiler.
//
// Callers are expected to (a) construct an Input from their project
// data, (b) call Compile with the target runtime name, and (c) call
// Tree.WriteTo to materialise the result on disk (or hand the Tree to
// packages/compile for Docker baking).
func Compile(runtimeName string, input Input) (*Tree, error) {
	r, err := lookupRuntime(runtimeName)
	if err != nil {
		return nil, err
	}
	return r.Render(input)
}
