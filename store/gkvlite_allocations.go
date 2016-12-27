package store

import (
	"errors"
	"github.com/spf13/viper"
)

func init() {
	StoreImpls["GKVLite"] = &gkvLiteFactory{}
}

type gkvLiteFactory struct{}

func (*gkvLiteFactory) Create(*viper.Viper) (AllocationStore, error) {
	return nil, errors.New("GVKLite not yet supported")
}
