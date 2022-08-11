package rproxy

// IfTrueElse is a helper that returns the argument based on whether or not it's true
func IfTrueElse[V any](condition bool, trueValue, falseValue V) V {
	if condition {
		return trueValue
	}
	return falseValue
}
