package db

type Driver interface {
	Save(v Digester) error
	LoadJSON(id string, v interface{}) error
}
