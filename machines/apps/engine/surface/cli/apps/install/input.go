package appsinstall

type Input struct{ Names []string }

func bind(in *Input, args []string) error {
	in.Names = args
	return nil
}
