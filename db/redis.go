package db

import (
	"encoding/json"
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/rkjdid/errors"
	"time"
)

type RedisDriver struct {
	Conn *redis.Client
}

var (
	KeyError = errors.Newf("key not found: %s")
)

func loadJSONResp(resp *redis.Resp, v interface{}) error {
	if resp.Err != nil {
		return resp.Err
	}
	b, err := resp.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (d *RedisDriver) LoadJSON(id string, v interface{}) error {
	resp := d.Conn.Cmd("GET", id)
	return loadJSONResp(resp, v)
}

func (d *RedisDriver) Save(v Digester) (id string, err error) {
	id, data, err := v.Digest()
	if err != nil {
		return id, err
	}
	return id, d.SET(id, data)
}

func (d *RedisDriver) SaveTTL(v Digester, ttl time.Duration) (id string, err error) {
	id, err = d.Save(v)
	if err == nil {
		err = d.EXPIRE(id, ttl)
	}
	return id, err
}

func (d *RedisDriver) SaveZScorer(item ZScorer, zkey string) (string, error) {
	id, err := d.Save(item)
	if err != nil {
		return "", err
	}
	return id, d.ZADD(zkey, id, item.ZScore())
}

func (d *RedisDriver) SET(id string, v interface{}) error {
	return d.Conn.Cmd("SET", id, v).Err
}

func (d *RedisDriver) ZADD(key string, id string, score float64) error {
	return d.Conn.Cmd("ZADD", key, score, id).Err
}

func (d *RedisDriver) ZRANK(hash string, id string) (int, error) {
	res := d.Conn.Cmd("ZRANK", hash, id)
	if res.Err != nil {
		return 0, res.Err
	}
	return res.Int()
}

func (d *RedisDriver) ZRANGE(key string, i, j int) ([]string, error) {
	resp := d.Conn.Cmd("ZRANGE", key, 0, -1)
	if resp.Err != nil {
		return nil, fmt.Errorf("zrange: %s", resp.Err)
	}
	return resp.List()
}

func (d *RedisDriver) EXPIRE(key string, ttl time.Duration) error {
	return d.Conn.Cmd("EXPIRE", key, int(ttl.Seconds())).Err
}
