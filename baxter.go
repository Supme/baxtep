package baxtep

import (
	"database/sql"
	"fmt"
)

type Baxtep struct {
	db database
}

type database struct {
	conn   *sql.DB
	driver string
	prefix string
}

func NewBaxtep(db *sql.DB, dbType, prefix string) *Baxtep {
	return &Baxtep{
		db: database{
			conn:   db,
			driver: dbType,
			prefix: prefix,
		},
	}
}

var dbVersion = 1

func (b *Baxtep) InitDB() error {
	//_ = b.db.conn.QueryRow("SELECT Value FROM "+b.db.prefix+"_param WHERE Name='db_version'").Scan(&version)
	//if version == "" {
	//	version = "0"
	//}
	tx, err := b.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch b.db.driver {
	case "mysql", "tidb":
		var query = []string{
			"SET SQL_MODE = \"NO_AUTO_VALUE_ON_ZERO\";",
			"SET time_zone = \"+00:00\";",
			fmt.Sprintf(
				"CREATE TABLE IF NOT EXISTS `%s` ("+
				" `id` int(11) NOT NULL AUTO_INCREMENT,"+
				" `name` varchar(100) NOT NULL,"+
				" `password` varchar(64) NOT NULL,"+
				" `email` varchar(100) NOT NULL,"+
				" `registration_time` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',"+
				" `enable` tinyint(1) NOT NULL,"+
				" `session_id` varchar(64) NOT NULL,"+
				" `session_time` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',"+
				" PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
				b.db.prefix),
			fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s_param` ("+
				" `id` int(11) NOT NULL AUTO_INCREMENT,"+
				" `user_id` int(11) NOT NULL,"+
				" `key` varchar(100) NOT NULL,"+
				" `val` varchar(250) NOT NULL,"+
				" PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
				b.db.prefix),
		}
		for i := range query {
			_, err = tx.Exec(query[i])
			if err != nil {
				return err
			}
		}
	case "ql", "ql-mem":
		var query = []string{
			fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (" +
				" name string," +
				" password string," +
				" email string," +
				" registration_time time," +
				" enable bool," +
				" session_id string," +
				" session_time time" +
				");",
				b.db.prefix),
			fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_param (" +
				" user_id int," +
				" key string," +
				" val string" +
				");",
				b.db.prefix),
		}
		for i := range query {
			_, err = tx.Exec(query[i])
			if err != nil {
				return err
			}
		}
		err = tx.Commit()
	default:
		err = fmt.Errorf("Database type '%s' not supported", b.db.driver)
	}

	return err
}

func (b *Baxtep) AddNewUser(name, email string) (User, string, error) {
	u := User{db: b.db, Name: name, Email: email, Enable: false}
	err := b.CheckExistUserName(name)
	if err != nil {
		return User{}, "", err
	}
	err = b.CheckExistUserEmail(email)
	if err != nil {
		return User{}, "", err
	}
	confirm := generateRandomString(32)
	tx, err := b.db.conn.Begin()
	if err != nil {
		return User{}, "", nil
	}
	defer tx.Rollback()
	var res sql.Result
	switch b.db.driver {
	case "mysql":
		res, err = tx.Exec("INSERT INTO `"+b.db.prefix+"`(`name`, `email`, `session_id`, `enable`) VALUES (?, ?, ?, FALSE)", u.Name, u.Email, confirm)
	case "ql", "ql-mem":
		res, err = tx.Exec("INSERT INTO "+b.db.prefix+"(name, email, session_id, enable) VALUES ($1, $2, $3, false)", u.Name, u.Email, confirm)
	}
	if err != nil {
		return User{}, "", err
	}
	u.id, err = res.LastInsertId()
	if err != nil {
		return User{}, "", err
	}
	err = tx.Commit()
	return u, confirm, err
}

func (b *Baxtep) ConfirmRegistration(str string) (User, error) {
	user, err := b.GetUserBySessionID(str)
	if err == ErrUserSessionNotFound {
		return user, ErrUserNotFound
	}
	err = user.SetEnable()
	return user, err
}

func (b *Baxtep) GetUserByEmail(email string) (User, error) {
	var row *sql.Row
	u := User{db: b.db, Email: email}
	switch b.db.driver {
	case "mysql":
		row = b.db.conn.QueryRow("SELECT `id`, `name`, `enable` FROM `"+b.db.prefix+"` WHERE `email`=?", u.Email)
	case "ql", "ql-mem":
		row = b.db.conn.QueryRow("SELECT id(), name, enable FROM "+b.db.prefix+" WHERE email=$1", u.Email)
	}
	err := row.Scan(&u.id, &u.Name, &u.Enable)
	if err == sql.ErrNoRows {
		return u, ErrUserWithEmailNotFound
	}
	return u, err
}

func (b *Baxtep) GetUserByName(name string) (User, error) {
	u := User{db: b.db, Name: name}
	var row *sql.Row
	switch b.db.driver {
	case "mysql":
		row = b.db.conn.QueryRow("SELECT `id`, `email`, `enable` FROM `"+b.db.prefix+"` WHERE `name`=?", u.Name)
	case "ql", "ql-mem":
		row = b.db.conn.QueryRow("SELECT id(), email, enable FROM "+b.db.prefix+" WHERE name=$1", u.Name)
	}
	err := row.Scan(&u.id, &u.Email, &u.Enable)
	if err == sql.ErrNoRows {
		err = ErrUserWithNameNotFound
	}
	return u, err
}

func (b *Baxtep) GetUserByID(id int64) (User, error) {
	u := User{db: b.db, id: id}
	var row *sql.Row
	switch b.db.driver {
	case "mysql":
		row = b.db.conn.QueryRow("SELECT `name`, `email`, `enable` FROM `"+b.db.prefix+"` WHERE `id`=?", u.id)
	case "ql", "ql-mem":
		row = b.db.conn.QueryRow("SELECT name, email, enable FROM "+b.db.prefix+" WHERE id()=?", u.id)
	}
	err := row.Scan(&u.Name, &u.Email, &u.Enable)
	if err == sql.ErrNoRows {
		err = ErrUserWithIDNotFound
	}
	return u, err
}

func (b *Baxtep) GetUserBySessionID(sessionID string) (User, error) {
	u := User{db: b.db}
	var row *sql.Row
	switch b.db.driver {
	case "mysql":
		row = b.db.conn.QueryRow("SELECT `id`, `name`, `email`, `enable` FROM `"+b.db.prefix+"` WHERE `session_id`=?", sessionID)
	case "ql", "ql-mem":
		row = b.db.conn.QueryRow("SELECT id(), name, email, enable FROM "+b.db.prefix+" WHERE session_id=$1", sessionID)
	}

	err := row.Scan(&u.id, &u.Name, &u.Email, &u.Enable)
	if err == sql.ErrNoRows {
		err = ErrUserSessionNotFound
	}
	return u, err
}

func (b *Baxtep) GetUserByEmailPassword(email, password string) (User, error) {
	u, err := b.GetUserByEmail(email)
	if err != nil {
		return u, err
	}
	err = u.CheckPassword(password)
	return u, err
}

func (b *Baxtep) GetUserByUsernamePassword(name, password string) (User, error) {
	u, err := b.GetUserByName(name)
	if err != nil {
		return u, err
	}
	err = u.CheckPassword(password)
	return u, err
}

func (b *Baxtep) CheckExistUserName(username string) error {
	var count int64
	var row *sql.Row
	switch b.db.driver {
	case "mysql":
		row = b.db.conn.QueryRow("SELECT COUNT(*) FROM "+b.db.prefix+" WHERE name=?", username)
	case "ql", "ql-mem":
		row = b.db.conn.QueryRow("SELECT count(*) FROM "+b.db.prefix+" WHERE name=$1", username)
	}
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	if count != 0 {
		return ErrUserNameExist
	}
	return nil
}

func (b *Baxtep) CheckExistUserEmail(email string) error {
	return checkExistUserEmail(b.db, email)
}

func (b *Baxtep) DeleteUser(id int64) error {
	tx, err := b.db.conn.Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	switch b.db.driver {
	case "mysql":
		_, err = b.db.conn.Exec("DELETE FROM `"+b.db.prefix+"` WHERE `id`=?", id)
	case "ql", "ql-mem":
		_, err = b.db.conn.Exec("DELETE FROM "+b.db.prefix+" WHERE id()=$1", id)
	}
	tx.Commit()
	return err
}
