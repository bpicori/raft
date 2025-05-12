package application

import "strconv"

func loadAndConvertToInt32(key string) (int32, error) {
	value, ok := hashMap.Load(key)
	if !ok {
		return 0, nil
	}

	currentValueInt, err := strconv.Atoi(value.(string))
	if err != nil {
		return 0, err
	}

	return int32(currentValueInt), nil
}
