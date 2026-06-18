package transform

func TransformLine(line, format string) (string, error) {
	switch format {
	case "rule":
		return tryTransform(line, TransformRuleLine, TransformQxLine)
	case "set":
		return tryTransform(line, TransformSetLine, TransformRuleLine, TransformQxLine)
	case "qx", "":
		return tryTransform(line, TransformQxLine, TransformRuleLine)
	default:
		return tryTransform(line, TransformQxLine, TransformRuleLine, TransformSetLine)
	}
}

func tryTransform(line string, fns ...func(string) (string, error)) (string, error) {
	var err error
	for _, fn := range fns {
		if result, e := fn(line); e == nil {
			return result, nil
		} else if err == nil {
			err = e
		}
	}
	if err == nil {
		err = errUnrecognized
	}
	return "", err
}
