package contract

type Layouts interface {
	// invariant: act runs with the project at root laid out as its
	// spwn.yaml says, and its error comes back as it is; a layout that does
	// not read is *project.LayoutRefused, and act never runs. An empty root
	// is the user's own folders, read with no layout.
	Within(root string, act func() error) error
}
