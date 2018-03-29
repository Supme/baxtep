package baxtep

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

type User struct {
	db      database
	id      int64
	Name    string
	Email   string
	Enable bool
}

func checkExistUserEmail(db database, email string) error {
	var count int64
	var row *sql.Row
	switch db.driver {
	case "mysql":
		row = db.conn.QueryRow("SELECT COUNT(*) FROM `"+db.prefix+"` WHERE `email`=?", email)
	case "ql", "ql-mem":
		row = db.conn.QueryRow("SELECT count(*) FROM "+db.prefix+" WHERE `email`=$1", email)
	}
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	if count != 0 {
		return ErrUserEmailExist
	}
	return nil
}

func (u *User) GetID() int64 {
	return u.id
}

func (u *User) setEnabled(enable bool) error {
	tx, err := u.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch u.db.driver {
	case "mysql":
		_, err = tx.Exec("UPDATE `"+u.db.prefix+"` SET `enable`=? WHERE id=?", enable, u.id)
	case "ql", "ql-mem":
		_, err = tx.Exec("UPDATE "+u.db.prefix+" SET enable=$1 WHERE id()=$2", enable, u.id)
	}
	if err != nil {
		return err
	}
	u.Enable = false
	return tx.Commit()
}

func (u *User) SetEnable() error {
	return u.setEnabled(true)
}

func (u *User) SetDisable() error {
	return u.setEnabled(false)
}

func (u *User) CheckPassword(password string) error {
	var (
		passhash string
		row *sql.Row
	)
	switch u.db.driver {
	case "mysql":
		row = u.db.conn.QueryRow("SELECT `password` FROM `"+u.db.prefix+"` WHERE `id`=?", u.id)
	case "ql", "ql-mem":
		row = u.db.conn.QueryRow("SELECT password FROM "+u.db.prefix+" WHERE id()=$1", u.id)
	}
	err := row.Scan(&passhash)
	if err != nil {
		return err
	}
	if getPasswordHash(password) != passhash {
		return ErrUserBadPassword
	}
	return nil
}

func (u *User) GetUpdate() error {
	var row *sql.Row
	switch u.db.driver {
	case "mysql":
		row = u.db.conn.QueryRow("SELECT `name`, `email`, `enable` FROM `"+u.db.prefix+"` WHERE `id`=?", u.id)
	case "ql", "ql-mem":
		row = u.db.conn.QueryRow("SELECT name, email, enable FROM "+u.db.prefix+" WHERE id()=$1", u.id)
	}
	return row.Scan(&u.Name, &u.Email, &u.Enable)
}

func (u *User) SetNewEmail(email string) error {
	err := checkExistUserEmail(u.db, email)
	if err != nil {
		return err
	}
	tx, err := u.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch u.db.driver {
	case "mysql":
		_, err = u.db.conn.Exec("UPDATE `"+u.db.prefix+"` SET `email`=? WHERE id=?", email, u.id)
	case "ql", "ql-mem":
		_, err = u.db.conn.Exec("UPDATE "+u.db.prefix+" SET email=$1 WHERE id()=$2", email, u.id)
	}
	if err != nil {
		return err
	}
	u.Email = email
	return tx.Commit()
}

func (u *User) SetNewPassword(password string) error {
	passhash := getPasswordHash(password)
	tx, err := u.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch u.db.driver {
	case "mysql":
		_, err = tx.Exec("UPDATE `"+u.db.prefix+"` SET `password`=? WHERE `id`=?", passhash, u.id)
	case "ql", "ql-mem":
		_, err = tx.Exec("UPDATE "+u.db.prefix+" SET password=$1 WHERE id()=$2", passhash, u.id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (u *User) GetNewPassword() (string, error) {
	password := generateRandomString(8)
	err := u.SetNewPassword(password)
	return password, err
}

func (u *User) CheckSessionID(expiredDuration time.Duration) error {
	var (
		sessionTime      time.Time
		mysqlSessionTime mysql.NullTime
		query string
	)
	switch u.db.driver {
	case "mysql":
		query = "SELECT `session_time` FROM `"+u.db.prefix+"` WHERE `id`=?"
	case "ql", "ql-mem":
		query = "SELECT session_time FROM "+u.db.prefix+" WHERE id()=$1"
	}
	row := u.db.conn.QueryRow(query, u.id)
	err := row.Scan(&sessionTime)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if mysqlSessionTime.Valid {
		sessionTime = mysqlSessionTime.Time
	}
	if time.Now().UTC().After(sessionTime.Add(expiredDuration)) {
		return ErrUserSessionExpired
	}
	return nil
}

func (u *User) SetNewSessionID() (string, error) {
	var err error
	sessionID := generateRandomString(64)
	tx, err := u.db.conn.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	switch u.db.driver {
	case "mysql":
		_, err = tx.Exec("UPDATE `"+u.db.prefix+"` SET `session_id`=?, `session_time`=? WHERE `id`=?", sessionID, time.Now(), u.id)
	case "ql", "ql-mem":
		_, err = tx.Exec("UPDATE "+u.db.prefix+" SET session_id=$1, session_time=$2 WHERE id()=$3", sessionID, time.Now(), u.id)
	}
	if err != nil {
		return "", err
	}
	err = tx.Commit()
	return sessionID, err
}

func (u *User) AddParams(params ...map[string]string) error {
	tx, err := u.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i := range params {
		for k, v := range params[i] {
			switch u.db.driver {
			case "mysql":
				_, err = tx.Exec("INSERT INTO `"+u.db.prefix+"_param` (`user_id`, `key`, `val`) VALUES (?, ?, ?)", u.id, k, v)
			case "ql", "ql-mem":
				_, err = tx.Exec("INSERT INTO "+u.db.prefix+"_param (user_id, key, val) VALUES ($1, $2, $3)", u.id, k, v)
			}
			if err != nil {
				return nil
			}
		}
	}
	return tx.Commit()
}

func (u *User) UpdateParams(key string, value ...string) error {
	err := u.DeleteParams(key)
	if err != nil {
		return err
	}
	var params []map[string]string
	for i := range value {
		params = append(params, map[string]string{key: value[i]})
	}
	return u.AddParams(params...)
}

func (u *User) HasParam(key string) (bool, error) {
	var (
		cnt int64
		err error
	)
	switch u.db.driver {
	case "mysql":
		err = u.db.conn.QueryRow("SELECT COUNT(*) FROM `"+u.db.prefix+"_param` WHERE `user_id`=? AND `key`=?", u.id, key).Scan(&cnt)
	case "ql", "ql-mem":
		err = u.db.conn.QueryRow("SELECT count(*) FROM "+u.db.prefix+"_param WHERE user_id=$1 AND key=$2", u.id, key).Scan(&cnt)
	}
	return cnt > 0, err
}

func (u *User) HasParamValue(key, value string) (bool, error) {
	var (
		cnt int64
		err error
	)
	switch u.db.driver {
	case "mysql":
		err = u.db.conn.QueryRow("SELECT COUNT(*) FROM `"+u.db.prefix+"_param` WHERE `user_id`=? AND `key`=? AND `val`=?", u.id, key, value).Scan(&cnt)
	case "ql", "ql-mem":
		err = u.db.conn.QueryRow("SELECT count(*) FROM "+u.db.prefix+"_param WHERE user_id=$1 AND key=$2 AND val=$3", u.id, key, value).Scan(&cnt)
	}
	return cnt > 0, err
}

func (u *User) GetParam(key string) ([]string, error) {
	var (
		param []string
		err   error
	)
	var rows *sql.Rows
	switch u.db.driver {
	case "mysql":
		rows, err = u.db.conn.Query("SELECT `val` FROM `"+u.db.prefix+"_param` WHERE `user_id`=? AND `key`=?", u.id, key)
	case "ql", "ql-mem":
		rows, err = u.db.conn.Query("SELECT val FROM "+u.db.prefix+"_param WHERE user_id=$1 AND key=$2", u.id, key)
	}
	if err != nil {
		return param, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		err = rows.Scan(&v)
		if err != nil {
			return param, err
		}
		param = append(param, v)
	}
	return param, nil
}

func (u *User) GetParams() (map[string][]string, error) {
	var (
		params = map[string][]string{}
		rows *sql.Rows
		err    error
	)
	switch u.db.driver {
	case "mysql":
		rows, err = u.db.conn.Query("SELECT `key`, `val` FROM `"+u.db.prefix+"_param` WHERE `user_id`=?", u.id)
	case "ql", "ql-mem":
		rows, err = u.db.conn.Query("SELECT key, val FROM "+u.db.prefix+"_param WHERE user_id=$1", u.id)
	}
	if err != nil {
		return params, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		err = rows.Scan(&k, &v)
		if err != nil {
			return params, err
		}
		if _, ok := params[k]; !ok {
			params[k] = []string{}
		}
		params[k] = append(params[k], v)
	}
	return params, nil
}

func (u *User) DeleteParams(keys ...string) error {
	var (
		placeholders string
		params       []interface{}
	)
	tx, err := u.db.conn.Begin()
	if err !=nil {
		return nil
	}
	defer tx.Rollback()
	switch u.db.driver {
	case "mysql":
		placeholders = strings.TrimLeft(strings.Repeat(", ?", len(keys)), ", ")
		for i := range keys {
			params = append(params, keys[i])
		}
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM `"+u.db.prefix+"_param` WHERE `user_id`=%d AND `key` IN (%s)", u.id, placeholders), params...)
	case "ql", "ql-mem":
		var placeholders string
		for i := range keys {
			params = append(params, keys[i])
			placeholders += fmt.Sprintf("$%d, ", i + 1)
		}
		placeholders = placeholders[:len(placeholders)-2]
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM "+u.db.prefix+"_param WHERE user_id=%d AND key IN (%s)", u.id, placeholders), params...)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
