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

func (d *RedisDriver) Save(v Digester) (id string, err error) {
	id, data, err := v.Digest()
	if err != nil {
		return id, err
	}
	// set value
	return id, d.Conn.Cmd("SET", id, data).Err
}

func (d *RedisDriver) SaveZScorer(z ZScorer, key string) error {
	id, err := d.Save(z)
	if err != nil {
		return err
	}
	return d.ZADD(key, id, z.ZScore())
}

func (d *RedisDriver) ZADD(key string, id string, score float64) error {
	return d.Conn.Cmd("ZADD", key, score, id).Err
}
