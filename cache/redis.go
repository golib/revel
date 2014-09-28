package cache

import (
	"github.com/garyburd/redigo/redis"

	"time"
)

// Wraps the Redis client to meet the Cache interface.
type RedisCache struct {
	pool              *redis.Pool
	defaultExpiration time.Duration
}

// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCache(host string, password string, defaultExpiration time.Duration) RedisCache {
	var pool = &redis.Pool{
		MaxIdle:     5,
		MaxActive:   int(7 * 24 * time.Hour),
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// redis protocol
			protocol := "tcp"

			conn, err := redis.Dial(protocol, host)
			if err != nil {
				return nil, err
			}

			if len(password) > 0 {
				if _, err := conn.Do("AUTH", password); err != nil {
					conn.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err := conn.Do("PING"); err != nil {
					conn.Close()
					return nil, err
				}
			}

			return conn, err
		},
		// custom connection test method
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if _, err := conn.Do("PING"); err != nil {
				return err
			}

			return nil
		},
	}

	return RedisCache{pool, defaultExpiration}
}

func (c RedisCache) Set(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	return c.invoke(conn.Do, key, value, expires)
}

func (c RedisCache) Add(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	existed, err := exists(conn, key)
	if err != nil {
		return err
	}
	if existed {
		return ErrNotStored
	}

	return c.invoke(conn.Do, key, value, expires)
}

func (c RedisCache) Replace(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	existed, err := exists(conn, key)
	if err != nil {
		return err
	}
	if !existed {
		return ErrNotStored
	}

	err = c.invoke(conn.Do, key, value, expires)
	if value == nil {
		return ErrNotStored
	}

	return err
}

func (c RedisCache) Get(key string, ptrValue interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	raw, err := conn.Do("GET", key)
	if err != nil {
		return err
	}
	if raw == nil {
		return ErrCacheMiss
	}

	item, err := redis.Bytes(raw, err)
	if err != nil {
		return err
	}

	return Deserialize(item, ptrValue)
}

func (c RedisCache) GetMulti(keys ...string) (Getter, error) {
	conn := c.pool.Get()
	defer conn.Close()

	normalizeKeys := func(keys []string) []interface{} {
		result := make([]interface{}, len(keys))
		for i, key := range keys {
			result[i] = key
		}

		return result
	}

	items, err := redis.Values(conn.Do("MGET", normalizeKeys(keys)...))
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, ErrCacheMiss
	}

	key2val := make(map[string][]byte)
	for i, key := range keys {
		key2val[key] = nil

		if i < len(items) {
			val, ok := items[i].([]byte)
			if ok {
				key2val[key] = val
			}
		}
	}

	return RedisItemMapGetter(key2val), nil
}

func (c RedisCache) Delete(key string) error {
	conn := c.pool.Get()
	defer conn.Close()

	existed, err := redis.Bool(conn.Do("DEL", key))
	if err == nil && !existed {
		return ErrCacheMiss
	}

	return err
}

func (c RedisCache) Increment(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer conn.Close()

	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that. Since we need to do increment
	// ourselves instead of natively via INCRBY (redis doesn't support wrapping), we get the value
	// and do the exists check this way to minimize calls to Redis
	val, err := conn.Do("GET", key)
	if err != nil {
		return 0, err
	}
	if val == nil {
		return 0, ErrCacheMiss
	}

	valInt64, err := redis.Int64(val, nil)
	if err != nil {
		return 0, err
	}

	var result int64 = valInt64 + int64(delta)
	if _, err := conn.Do("SET", key, result); err != nil {
		return 0, err
	}

	return uint64(result), nil
}

func (c RedisCache) Decrement(key string, delta uint64) (newValue uint64, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that, hence the exists call
	existed, err := exists(conn, key)
	if err != nil {
		return 0, err
	}
	if !existed {
		return 0, ErrCacheMiss
	}

	// Decrement contract says you can only go to 0
	// so we go fetch the value and if the delta is greater than the amount,
	// 0 out the value
	valInt64, err := redis.Int64(conn.Do("GET", key))
	if err != nil {
		return 0, err
	}

	if delta > uint64(valInt64) {
		tempint, err := redis.Int64(conn.Do("DECRBY", key, valInt64))

		return uint64(tempint), err
	}

	tempint, err := redis.Int64(conn.Do("DECRBY", key, delta))
	return uint64(tempint), err
}

func (c RedisCache) Flush() error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("FLUSHALL")
	return err
}

func (c RedisCache) invoke(f func(string, ...interface{}) (interface{}, error),
	key string, value interface{}, expires time.Duration) error {

	switch expires {
	case DEFAULT:
		expires = c.defaultExpiration
	case FOREVER:
		expires = time.Duration(0)
	}

	b, err := Serialize(value)
	if err != nil {
		return err
	}

	conn := c.pool.Get()
	defer conn.Close()

	if expires > 0 {
		_, err := f("SETEX", key, int32(expires/time.Second), b)
		return err
	}

	_, err = f("SET", key, b)
	return err
}

func exists(conn redis.Conn, key string) (bool, error) {
	return redis.Bool(conn.Do("EXISTS", key))
}

// Implement a Getter on top of the returned item map.
type RedisItemMapGetter map[string][]byte

func (g RedisItemMapGetter) Get(key string, ptrValue interface{}) error {
	item, ok := g[key]
	if !ok {
		return ErrCacheMiss
	}

	return Deserialize(item, ptrValue)
}
