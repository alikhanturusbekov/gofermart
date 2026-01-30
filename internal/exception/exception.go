package exception

import (
	"errors"
	"fmt"
)

var (
	ErrRecordExists = errors.New("record already exists")

	ErrOrderExistsByUser  = fmt.Errorf("order already exists for this user")
	ErrOrderExistsByOther = fmt.Errorf("order already exists for another user")

	ErrNotEnoughBalance = errors.New("not enough balance")
)
