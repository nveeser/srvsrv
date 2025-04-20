package ctxerr

//go:noinline
func myFunc1(e error) error {
	var t testType
	return t.myFunc2(e)
}

type testType struct{}

//go:noinline
func (testType) myFunc2(e error) error {
	return myFunc3(e)
}

//go:noinline
func myFunc3(e error) error {
	return E(Op("myFunc3-operation"), e)
}
