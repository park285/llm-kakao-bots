package ptr

// String: 문자열 포인터를 만든다.
func String(v string) *string { return &v }

// Int: int 포인터를 만든다.
func Int(v int) *int { return &v }

// Int64 는 int64 포인터를 만든다.
func Int64(v int64) *int64 { return &v }

// Bool: bool 포인터를 만든다.
func Bool(v bool) *bool { return &v }

// Float64 는 float64 포인터를 만든다.
func Float64(v float64) *float64 { return &v }
