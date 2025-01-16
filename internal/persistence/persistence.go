package persistence

import (
	"context"
	"errors"
)

var ErrDuplicatedSIP = errors.New("there is already a SIP with the same checksum")

type Service interface {
	CreateSIP(context.Context, string, string) error
}
