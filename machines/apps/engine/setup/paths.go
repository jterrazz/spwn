package setup

type Roots struct{ Prefs, Data, State, Cache string }

type Paths struct {
	Prefs, Data, State, Cache, Bin string
	Roots                          Roots
}
