package db

import "fmt"

type NopDriver struct{}

func (NopDriver) Save(v Digester) (err error) { return nil }

func (NopDriver) LoadJSON(id string, v interface{}) error {
	return fmt.Errorf("LoadJSON unimplemented on NopDriver")
}
