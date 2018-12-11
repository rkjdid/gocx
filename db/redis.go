package db

import (
	"encoding/json"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"time"
)

type RedisDriver struct {
	Pool *pool.Pool
}

func (d *RedisDriver) LoadJSON(id string, v interface{}) error {
	resp := d.Pool.Cmd("GET", id)
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
	return d.Pool.Cmd("SET", id, v).Err
}

func (d *RedisDriver) ZADD(key string, id string, score float64) error {
	return d.Pool.Cmd("ZADD", key, score, id).Err
}

func (d *RedisDriver) ZRANK(hash string, id string) (int, error) {
	return d.cmdInt("ZRANK", hash, id)
}

func (d *RedisDriver) ZREVANK(hash string, id string) (int, error) {
	return d.cmdInt("ZREVANK", hash, id)
}

func (d *RedisDriver) ZRANGE(key string, i, j int) ([]string, error) {
	return d.cmdList("ZRANGE", key, 0, j)
}

func (d *RedisDriver) ZREVRANGE(key string, i, j int) ([]string, error) {
	return d.cmdList("ZREVRANGE", key, 0, j)
}

func (d *RedisDriver) EXPIRE(key string, ttl time.Duration) error {
	return d.Pool.Cmd("EXPIRE", key, int(ttl.Seconds())).Err
}

func (d *RedisDriver) cmdInt(cmd string, args ...interface{}) (int, error) {
	res := d.Pool.Cmd(cmd, args...)
	if res.Err != nil {
		return 0, res.Err
	}
	return res.Int()
}

func (d *RedisDriver) cmdList(cmd string, args ...interface{}) ([]string, error) {
	res := d.Pool.Cmd(cmd, args...)
	if res.Err != nil {
		return nil, res.Err
	}
	return res.List()
}

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
