package sub

import (
	"github.com/shopspring/decimal"
)

func AddNum(numA string, numB string) (string, error) {
	var decimalA, errA = decimal.NewFromString(numA)
	var decimalB, errB = decimal.NewFromString(numB)
	if errA != nil {
		return "", errA
	} else if errB != nil {
		return "", errB
	}
	var addNum = decimalA.Add(decimalB)
	return addNum.String(), nil
}
