package interfaces

// ToInterfaceSlice 任意切片转为[]interface{}
func ToInterfaceSlice[T any](slice []T) []interface{} {
	res := make([]interface{}, len(slice))
	for i, v := range slice {
		res[i] = v
	}
	return res
}
