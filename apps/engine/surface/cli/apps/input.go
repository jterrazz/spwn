package appscmd

type Input struct {
	Name string
	All  bool
}

func bind(in *Input, args []string) error {
	if len(args) > 0 {
		in.Name = args[0]
	}
	return nil
}
