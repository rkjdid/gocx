package db

type Driver interface {
	Save(v Digester) (err error)
	LoadJSON(id string, v interface{})
}
