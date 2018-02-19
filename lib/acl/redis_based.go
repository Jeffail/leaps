/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, sub to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package acl

import (
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/Jeffail/leaps/lib/util/service/log"
)

//--------------------------------------------------------------------------------------------------

// RedisConfig - A config object for the redis authentication object.
type RedisConfig struct {
	URL          string `json:"url" yaml:"url"`
	Password     string `json:"password" yaml:"password"`
	PoolIdleTOut int64  `json:"pool_idle_s" yaml:"pool_idle_s"`
	PoolMaxIdle  int    `json:"pool_max_idle" yaml:"pool_max_idle"`
}

// NewRedisConfig - Returns a default config object for a Redis.
func NewRedisConfig() RedisConfig {
	return RedisConfig{
		URL:          ":6379",
		Password:     "",
		PoolIdleTOut: 240,
		PoolMaxIdle:  3,
	}
}

//--------------------------------------------------------------------------------------------------

func newPool(config RedisConfig) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     config.PoolMaxIdle,
		IdleTimeout: time.Duration(config.PoolIdleTOut) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", config.URL)
			if err != nil {
				return nil, err
			}
			if 0 != len(config.Password) {
				if _, err := c.Do("AUTH", config.Password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

//--------------------------------------------------------------------------------------------------

// Errors for the Redis type.
var (
	ErrNoKey = errors.New("key did not exist")
)

/*
Redis - An Authenticator type that uses Redis for passing authentication tokens into leaps.

Leaps can be configured to use Redis for authentication. A service can provide an access token to
leaps on behalf of a prospective user by adding a single key to the shared redis instance that
outlines the type of access to be provided.

The key should be the unique, one-use access token that is also shared with the user. The value is a
JSON blob that details the credentials and access level of the authentication:

Key:   <token>
Value: { "access_level":"<access_level>", "user_id":"<user_id>", "document_id":"<document_id>" }

<token>        The shared token that the user is also given and subsequently provides to leaps.
<access_level> The access level that your service wishes to grant the user.
<user_id>      The id of the authenticated user.
<document_id>  The id of the document, omit or leave this blank if you are granting CREATE access.

The options for <access_level> are `CREATE`, `EDIT` and `READ`.

Once leaps has read and verified the auth token it will delete the key. Key/value pairs should have
a TTL such that they will expire if not used.
*/
type Redis struct {
	logger log.Modular
	config RedisConfig
	pool   *redis.Pool
}

// NewRedis - Creates a Redis using the provided configuration.
func NewRedis(config RedisConfig, logger log.Modular) Authenticator {
	return &Redis{
		logger: logger.NewModule(":redis_auth"),
		config: config,
		pool:   newPool(config),
	}
}

//--------------------------------------------------------------------------------------------------

// Authenticate - Reads a key (token) from redis and parses the value to check for an access level.
func (s *Redis) Authenticate(userMetadata interface{}, token, documentID string) AccessLevel {
	value, err := s.ReadKey(token)
	if err != nil {
		s.logger.Errorf("Failed to access token: %v\n", err)
		return NoAccess
	}

	credentials := struct {
		AccessLevel  string      `json:"access_level"`
		UserMetadata interface{} `json:"user_metadata"`
		DocumentID   string      `json:"document_id"`
	}{}

	if err := json.Unmarshal([]byte(value), &credentials); err != nil {
		s.logger.Errorf("Token value `%v` could not be parsed: %v\n", value, err)
		return NoAccess
	}

	accessLevel := NoAccess
	switch credentials.AccessLevel {
	case "CREATE":
		accessLevel = CreateAccess
	case "EDIT":
		accessLevel = EditAccess
	case "READ":
		accessLevel = ReadAccess
	}

	if accessLevel == NoAccess {
		s.logger.Errorf("Token value `%v` did not provide valid access level\n", value)
		return NoAccess
	}

	if len(credentials.DocumentID) <= 0 {
		s.logger.Errorf("Token value `%v` did not provide a valid document ID\n", value)
		return NoAccess
	}

	if !reflect.DeepEqual(credentials.UserMetadata, userMetadata) {
		s.logger.Warnf(
			"Incorrect user ID provided to authenticator, token contents: `%v`,  provided userMetadata: `%v`\n",
			value, userMetadata,
		)
		return NoAccess
	}

	if credentials.DocumentID != documentID {
		s.logger.Warnf(
			"Incorrect document ID provided to authenticator, token contents: `%v`,  provided documentID: `%v`\n",
			value, documentID,
		)
		return NoAccess
	}

	err = s.DeleteKey(token)
	if err != nil {
		s.logger.Errorf("failed to delete key: %v\n", token)
	}

	return accessLevel
}

// ReadKey - Simply return the value of a particular key, or an error.
func (s *Redis) ReadKey(key string) (string, error) {
	conn := s.pool.Get()
	defer conn.Close()

	reply, err := redis.String(conn.Do("GET", key))
	if err != nil {
		return "", err
	}
	return reply, nil
}

// DeleteKey - Deletes an existing key.
func (s *Redis) DeleteKey(key string) error {
	conn := s.pool.Get()
	defer conn.Close()

	reply, err := redis.Int(conn.Do("DEL", key))
	if err != nil {
		return err
	}
	if 0 == reply {
		return ErrNoKey
	}
	return nil
}

//--------------------------------------------------------------------------------------------------
