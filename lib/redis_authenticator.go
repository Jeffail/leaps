package lib

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jeffail/leaps/util"
	"github.com/youtube/vitess/go/pools"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
RedisAuthenticatorConfig - A config object for the redis authentication object.
*/
type RedisAuthenticatorConfig struct {
	URL string `json:"url"`
}

/*
DefaultRedisAuthenticatorConfig - Returns a default config object for a RedisAuthenticator.
*/
func DefaultRedisAuthenticatorConfig() RedisAuthenticatorConfig {
	return RedisAuthenticatorConfig{
		URL: ":6379",
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
resourceConn - adapts a Redigo connection to a Vitess Resource.
*/
type resourceConn struct {
	conn redis.Conn
}

/*
resourceConn - adapts a Redigo connection to a Vitess Resource.
*/
func (r resourceConn) Close() {
	r.conn.Close()
}

/*--------------------------------------------------------------------------------------------------
 */

/*
RedisAuthenticator - A wrapper around the Redis client that acts as an authenticator.
*/
type RedisAuthenticator struct {
	logger *util.Logger
	config TokenAuthenticatorConfig
	pool   *pools.ResourcePool
}

/*
CreateRedisAuthenticator - Creates a RedisAuthenticator using the provided configuration.
*/
func CreateRedisAuthenticator(config TokenAuthenticatorConfig, logger *util.Logger) *RedisAuthenticator {
	p := pools.NewResourcePool(func() (pools.Resource, error) {
		c, err := redis.Dial("tcp", config.RedisConfig.URL)
		return resourceConn{c}, err
	}, 10, 200, time.Minute)

	return &RedisAuthenticator{
		logger: logger.NewModule("[redis_auth]"),
		config: config,
		pool:   p,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
AuthoriseCreate - Checks whether a specific key exists in Redis and that the value matches our user
ID.
*/
func (s *RedisAuthenticator) AuthoriseCreate(token, userID string) bool {
	if !s.config.AllowCreate {
		return false
	}
	userKey, err := s.ReadKey(token)
	if err != nil {
		s.logger.Errorf("failed to get authorise create token: %v\n", err)
		return false
	}
	if userKey != userID {
		s.logger.Warnf("create token invalid, provided: %v, actual: %v\n", userID, userKey)
		return false
	}
	err = s.DeleteKey(token)
	if err != nil {
		s.logger.Errorf("failed to delete key: %v\n", token)
	}
	return true
}

/*
AuthoriseJoin - Checks whether a specific key exists in Redis and that the value matches a document
ID.
*/
func (s *RedisAuthenticator) AuthoriseJoin(token, documentID string) bool {
	docKey, err := s.ReadKey(token)
	if err != nil {
		s.logger.Errorf("failed to get authorise join token: %v\n", err)
		return false
	}
	if docKey != documentID {
		s.logger.Warnf("join token invalid, provided: %v, actual: %v\n", documentID, docKey)
		return false
	}
	err = s.DeleteKey(token)
	if err != nil {
		s.logger.Errorf("failed to delete key: %v\n", token)
	}
	return true
}

/*
ReadKey - Simply return the value of a particular key, or an error.
*/
func (s *RedisAuthenticator) ReadKey(key string) (string, error) {
	pItem, err := s.pool.Get()
	if err != nil {
		return "", err
	}
	defer s.pool.Put(pItem)
	redisConn := pItem.(resourceConn)

	reply, err := redis.String(redisConn.conn.Do("GET", key))
	if err != nil {
		return "", err
	}
	return reply, nil
}

/*
DeleteKey - Deletes an existing key.
*/
func (s *RedisAuthenticator) DeleteKey(key string) error {
	pItem, err := s.pool.Get()
	if err != nil {
		return err
	}
	defer s.pool.Put(pItem)
	redisConn := pItem.(resourceConn)

	reply, err := redis.Int(redisConn.conn.Do("DEL", key))
	if err != nil {
		return err
	}
	if 0 == reply {
		return errors.New("key did not exist")
	}
	return nil
}

/*--------------------------------------------------------------------------------------------------
 */
