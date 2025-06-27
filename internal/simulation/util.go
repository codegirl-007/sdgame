package simulation

func AsFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func AsString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
