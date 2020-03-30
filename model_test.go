package gom

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DBUser     = "root"
	DBPassword = "11111111"
	DBIP       = "127.0.0.1"
	DBPort     = "3306"
	Database   = "test"
	DBOptions  = "charset=utf8mb4&parseTime=True&loc=Local"
)

type User struct {
	ID          int64
	AccountName string
	NickName    string
	Password    string
	CreateTime  time.Time
	Data        []byte
}

func (u *User) Scan(scanable Scanable) error {
	return scanable.Scan(&u.ID, &u.AccountName, &u.NickName, &u.Password, &u.CreateTime, &u.Data)
}

func getMockUserList() []*User {
	return []*User{
		&User{AccountName: "admin", NickName: "admin", Password: "******", Data: []byte("i am admin")},
		&User{AccountName: "Aabbye", NickName: "ab", Password: "------", Data: []byte("Aabbye is smart")},
		&User{AccountName: "Cadence", NickName: "cad", Password: "$$$$$$", Data: []byte("Cadence like cat and cake")},
		&User{AccountName: "Galen Adams", NickName: "Adams", Password: "======", Data: []byte("Adams oh Adams")},
		&User{AccountName: "Palmira", NickName: "pm", Password: "@@@@@@", Data: []byte("Palmira is a good name")},
	}
}

func TestOrm(t *testing.T) {

	DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", DBUser, DBPassword, DBIP, DBPort, Database, DBOptions)
	db, err := sql.Open("mysql", DBURL)
	if err != nil {
		t.Error("open database error:", err)
		return
	}
	t.Logf("open db passed!")

	orm := NewOrm()
	sql := `select * from user`
	if rows, err := db.Query(sql); err != nil {
		t.Error("execute sql get error:", err)
	} else {
		if users, err := orm.ToMultiObjs(rows, &User{}); err != nil {
			t.Error("convert sql result get error:", err)
		} else {
			t.Log("queryAll passed!")
			for i, v := range users {
				t.Logf("%d. user: %v", i, v)
			}
		}
	}

	sql = `select * from user limit 1`
	if row := db.QueryRow(sql); row != nil {
		if user, err := orm.ToObj(row, &User{}); err != nil {
			t.Error("convert sql result get error:", err)
		} else {
			t.Log("queryRow passed! user: ", user)
		}
	}
}

func TestLoadModel(t *testing.T) {
	m, err := LoadModel("model_test.yaml", &User{})
	if err != nil {
		t.Error("load yaml file failed:", err)
	} else {
		t.Log("load model passed!")
	}

	DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", DBUser, DBPassword, DBIP, DBPort, Database, DBOptions)
	db, err := sql.Open("mysql", DBURL)
	if err != nil {
		t.Error("open database error:", err)
		return
	} else {
		t.Logf("open db passed!")
	}

	if _, _, err = m.Exec(db, "dropTable"); err != nil {
		t.Error("dropTable failed!", err)
	} else {
		t.Logf("dropTable passed!")
	}

	if _, _, err = m.Exec(db, "createTable"); err != nil {
		t.Error("createTable failed!", err)
	} else {
		t.Logf("createTable passed!")
	}

	n, id, err := m.MultiInsert(db, "insert", getMockUserList(), 2)
	if err != nil {
		t.Error("insert data failed!", err)
	} else {
		t.Logf("multi insert %d rows, last insert id is %d", n, id)
	}
	/*
		for _, u := range getMockUserList() {
			if _, _, err = m.Exec(db, "insert", u); err != nil {
				t.Error("insert data failed!", err)
			} else {
				t.Logf("insert data passed!")
			}
		}
	*/

	if users, err := m.Query(db, "queryAll"); err != nil {
		t.Error("queryAll failed:", err)
	} else {
		t.Log("queryAll passed!")
		for i, v := range users {
			t.Logf("%d. user: %v", i, v)
		}
	}

	user, err := m.QueryRow(db, "queryByName", User{AccountName: "Aabbye"})
	if err != nil {
		t.Error("queryByName failed:", err)
	} else {
		t.Log("query by name passed!")
		t.Logf("user: %v", user)
	}

	user, err = m.QueryRow(db, "select <colum> from <table> where account_name = ${accountname}", User{AccountName: "Aabbye"})
	if err != nil {
		t.Error("query by raw sql failed:", err)
	} else {
		t.Log("query by raw sql passed!")
		t.Logf("user: %v", user)
	}

	n, _, err = m.Exec(db, "deleteById", user)
	if err != nil {
		t.Error("deleteById failed:", err)
	} else {
		t.Log("deleteById passed!")
		t.Logf("delete %d row", n)
	}

	n, _, err = m.Exec(db, "clear")
	if err != nil {
		t.Error("clear failed:", err)
	} else {
		t.Log("clear passed!")
		t.Logf("clear table delete %d row", n)
	}

	t.Error("END")
}
