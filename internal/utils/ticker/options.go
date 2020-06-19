package ticker

type Option = func(*Ticker)

func SkipFirst(py *Ticker) {
	py.runFirst = false
}
