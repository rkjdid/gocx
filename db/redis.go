package db

import (
	"encoding/json"
	"github.com/mediocregopher/radix.v2/redis"
)

type RedisDriver struct {
	Conn *redis.Client
}

func (d *RedisDriver) LoadJSON(id string, v interface{}) error {
	resp := d.Conn.Cmd("GET", id)
	if resp.Err != nil {
		return resp.Err
	}
	b, err := resp.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (d *RedisDriver) Save(v Digester) (err error) {
	id, data, err := v.Digest()
	if err != nil {
		return err
	}
	// set value
	err = d.Conn.Cmd("SET", id, data).Err
	if err != nil {
		return err
	}
	// zadd in sorted set
	if x, ok := v.(ZScore); ok {
		return d.Conn.Cmd("ZADD", "results", x.ZScore(), id).Err
	}
	return nil
}
